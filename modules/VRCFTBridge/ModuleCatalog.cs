using System.Reflection;
using System.Runtime.Loader;
using System.Text.Json;
using System.Text.RegularExpressions;
using Microsoft.Extensions.Logging;
using VRCFaceTracking;

namespace VRCFTBridge;

/// <summary>
/// The VRCFaceTracking modules bundled with the bridge: one folder per
/// module under modules/, laid out as VRCFaceTracking installs them.
/// </summary>
public sealed partial class ModuleCatalog(string dir, JsonLogger logger)
{
    /// <summary>The driver ID; module IDs are derived from module.json.</summary>
    public const string DriverId = "VRCFTBridge";

    private static readonly JsonSerializerOptions ReadOptions = new() { PropertyNameCaseInsensitive = true };
    private static readonly JsonSerializerOptions WriteOptions = new() { PropertyNamingPolicy = JsonNamingPolicy.CamelCase };

    /// <summary>The module.json VRCFaceTracking writes next to an installed module.</summary>
    private sealed class InstalledModule
    {
        public Guid ModuleId { get; set; }
        public string? Version { get; set; }
        public string? AuthorName { get; set; }
        public string? ModuleName { get; set; }
        public string? ModuleDescription { get; set; }
        public string? UsageInstructions { get; set; }
        public string? ModulePageUrl { get; set; }
        public string? DllFileName { get; set; }
    }

    private sealed record Entry(string Id, string DllPath, InstalledModule Info);

    // Metadata and Support as the host's --info JSON spells them.
    private sealed record Metadata(string Name, string Description, string Author, string Version,
        string PageUrl, string UsageInstructions, string[] Icons);
    private sealed record Support(bool Eye, bool Expression);
    private sealed record Description(string Id, Metadata Metadata, Support Supported);
    private sealed record DriverDescription(string Kind, string Id, Metadata Metadata, Description[] Modules);

    /// <summary>
    /// Returns the driver's --info JSON: the driver itself, described by
    /// metadata.json, and every bundled module that can be loaded.
    /// </summary>
    public string DescribeDriver(string metadataPath)
    {
        var md = JsonSerializer.Deserialize<JsonElement>(File.ReadAllText(metadataPath));
        string Get(string key) => md.TryGetProperty(key, out var v) ? v.GetString() ?? "" : "";
        var driver = new Metadata(Get("name"), Get("description"), Get("author"), Get("version"),
            Get("pageUrl"), Get("usageInstructions"), []);

        var modules = new List<Description>();
        foreach (var entry in Entries())
        {
            // Supported is only known from the module itself.
            var context = new ModuleLoadContext(entry.DllPath, collectible: true);
            try
            {
                var module = Instantiate(context, entry.DllPath);
                if (module == null) continue;
                var (eye, expression) = module.Supported;
                var info = entry.Info;
                modules.Add(new Description(entry.Id,
                    new Metadata(info.ModuleName ?? entry.Id, info.ModuleDescription ?? "", info.AuthorName ?? "",
                        info.Version ?? "", info.ModulePageUrl ?? "", info.UsageInstructions ?? "", []),
                    new Support(eye, expression)));
            }
            catch (Exception e)
            {
                logger.LogWarning(e, "Skipping module {path}", entry.DllPath);
            }
            finally
            {
                context.Unload();
            }
        }
        return JsonSerializer.Serialize(new DriverDescription("driver", DriverId, driver, modules.ToArray()), WriteOptions);
    }

    /// <summary>Loads and instantiates the module with the given ID.</summary>
    public ExtTrackingModule? Load(string id)
    {
        var entry = Entries().FirstOrDefault(e => e.Id == id);
        if (entry == null) return null;
        var module = Instantiate(new ModuleLoadContext(entry.DllPath, collectible: false), entry.DllPath);
        if (module != null) module.Logger = logger.Named(module.GetType().Name);
        return module;
    }

    private IEnumerable<Entry> Entries()
    {
        if (!Directory.Exists(dir)) yield break;
        foreach (var folder in Directory.GetDirectories(dir).Order(StringComparer.Ordinal))
        {
            var info = new InstalledModule();
            var json = Path.Combine(folder, "module.json");
            if (File.Exists(json))
            {
                try
                {
                    info = JsonSerializer.Deserialize<InstalledModule>(File.ReadAllText(json), ReadOptions) ?? info;
                }
                catch (JsonException e)
                {
                    logger.LogWarning(e, "Ignoring invalid {path}", json);
                }
            }
            var dll = FindDll(folder, info.DllFileName);
            if (dll == null)
            {
                logger.LogWarning("No module assembly in {folder}", folder);
                continue;
            }
            yield return new Entry(IdOf(info, folder), dll, info);
        }
    }

    // As VRCFaceTracking does: the named assembly, else the only one.
    private static string? FindDll(string folder, string? name)
    {
        if (!string.IsNullOrEmpty(name) && File.Exists(Path.Combine(folder, name))) return Path.Combine(folder, name);
        var dlls = Directory.GetFiles(folder, "*.dll");
        return dlls.Length == 1 ? dlls[0] : null;
    }

    /// <summary>
    /// A stable host ID: the VRCFaceTracking module ID, or the folder name
    /// for a local module without one. Host IDs may only contain letters,
    /// digits, '.', '_' and '-'.
    /// </summary>
    private static string IdOf(InstalledModule info, string folder) =>
        info.ModuleId != Guid.Empty
            ? "VRCFT-" + info.ModuleId.ToString("D")
            : "VRCFT-" + InvalidIdChars().Replace(Path.GetFileName(folder), "_");

    [GeneratedRegex("[^A-Za-z0-9._-]")]
    private static partial Regex InvalidIdChars();

    // As VRCFaceTracking does: the first ExtTrackingModule in the assembly.
    private ExtTrackingModule? Instantiate(ModuleLoadContext context, string dllPath)
    {
        var assembly = context.LoadFromAssemblyPath(dllPath);
        Type?[] types;
        try
        {
            types = assembly.GetTypes();
        }
        catch (ReflectionTypeLoadException e)
        {
            types = e.Types;
        }
        var type = types.FirstOrDefault(t => t != null && !t.IsAbstract && t.IsSubclassOf(typeof(ExtTrackingModule)));
        if (type == null)
        {
            logger.LogWarning("{path} contains no ExtTrackingModule", dllPath);
            return null;
        }
        return (ExtTrackingModule?)Activator.CreateInstance(type);
    }
}

/// <summary>
/// Loads one module with its own dependencies. The VRCFaceTracking
/// assemblies and the logging abstractions come from the bridge, so the
/// module's types match the ones the bridge uses.
/// </summary>
public sealed class ModuleLoadContext(string dllPath, bool collectible)
    : AssemblyLoadContext(Path.GetFileName(dllPath), collectible)
{
    private static readonly HashSet<string> Shared =
        ["VRCFaceTracking.Core", "VRCFaceTracking.SDK", "Microsoft.Extensions.Logging.Abstractions"];

    private readonly AssemblyDependencyResolver _resolver = new(dllPath);
    private readonly string _dir = Path.GetDirectoryName(dllPath)!;

    protected override Assembly? Load(AssemblyName name)
    {
        if (name.Name == null || Shared.Contains(name.Name)) return null;
        var path = _resolver.ResolveAssemblyToPath(name) ?? Existing(Path.Combine(_dir, name.Name + ".dll"));
        return path != null ? LoadFromAssemblyPath(path) : null;
    }

    protected override IntPtr LoadUnmanagedDll(string name)
    {
        var path = _resolver.ResolveUnmanagedDllToPath(name) ?? Existing(Path.Combine(_dir, name));
        return path != null ? LoadUnmanagedDllFromPath(path) : IntPtr.Zero;
    }

    private static string? Existing(string path) => File.Exists(path) ? path : null;
}
