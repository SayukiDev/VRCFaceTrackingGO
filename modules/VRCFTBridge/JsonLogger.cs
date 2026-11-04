using System.Text.Json;
using Microsoft.Extensions.Logging;

namespace VRCFTBridge;

/// <summary>
/// Writes log entries as the JSON lines the host re-logs from a module's
/// stderr ({"level", "msg", "logger", ...}).
/// </summary>
public sealed class JsonLogger(string name) : ILogger
{
    private static readonly object WriteLock = new();
    private static readonly TextWriter Stderr = Console.Error;

    /// <summary>A logger for a sub-component, like zap's Named.</summary>
    public JsonLogger Named(string sub) => new($"{name}.{sub}");

    public IDisposable? BeginScope<TState>(TState state) where TState : notnull => null;

    public bool IsEnabled(LogLevel logLevel) => logLevel != LogLevel.None;

    public void Log<TState>(LogLevel logLevel, EventId eventId, TState state, Exception? exception, Func<TState, Exception?, string> formatter)
    {
        if (!IsEnabled(logLevel)) return;
        var entry = new Dictionary<string, object?>
        {
            ["level"] = Level(logLevel),
            ["msg"] = formatter(state, exception),
            ["logger"] = name,
        };
        if (exception != null) entry["error"] = exception.ToString();
        var line = JsonSerializer.Serialize(entry);
        lock (WriteLock)
        {
            Stderr.WriteLine(line);
            Stderr.Flush();
        }
    }

    private static string Level(LogLevel level) => level switch
    {
        LogLevel.Trace or LogLevel.Debug => "debug",
        LogLevel.Information => "info",
        LogLevel.Warning => "warn",
        _ => "error",
    };
}
