// VRCFTBridge is a VRCFaceTrackingGO driver program that runs VRCFaceTracking
// tracking modules (ExtTrackingModule implementations in .NET assemblies).
//
//   index.vfgmod --info                          lists the bundled modules
//   index.vfgmod --vfgmod=true --source=<id>     serves one of them
//
// Modules live in modules/<name>/ next to the program, each with the
// module.json VRCFaceTracking writes when it installs a module.
using System.Text;
using VRCFTBridge;

string? source = null;
bool info = false, host = false;
foreach (var arg in args)
{
    if (arg == "--info") info = true;
    else if (arg == "--vfgmod=true") host = true;
    else if (arg.StartsWith("--source=")) source = arg["--source=".Length..];
    else
    {
        host = false;
        break;
    }
}

// Only the protocol may use stdout; anything a module prints goes to stderr.
var stdout = Console.OpenStandardOutput();
Console.SetOut(Console.Error);

var logger = new JsonLogger("VRCFTBridge");
var catalog = new ModuleCatalog(Path.Combine(AppContext.BaseDirectory, "modules"), logger);

if (info)
{
    var json = catalog.DescribeDriver(Path.Combine(AppContext.BaseDirectory, "metadata.json"));
    stdout.Write(Encoding.UTF8.GetBytes(json + "\n"));
    return 0;
}
if (!host)
{
    Console.Error.WriteLine("VRCFTBridge is a VRCFaceTracking driver. Install it from the Modules page instead of running it.");
    return 2;
}
if (string.IsNullOrEmpty(source))
{
    Console.Error.WriteLine("no module selected: use --source");
    return 2;
}

var module = catalog.Load(source);
if (module == null)
{
    Console.Error.WriteLine($"this driver contains no loadable module \"{source}\"");
    return 2;
}
return new Server(module, logger).Serve(Console.OpenStandardInput(), stdout);
