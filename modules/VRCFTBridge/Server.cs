using System.Buffers.Binary;
using System.Text;
using Microsoft.Extensions.Logging;
using VRCFaceTracking;
using VRCFaceTracking.Core.Library;

namespace VRCFTBridge;

/// <summary>
/// Answers the host's requests for one VRCFaceTracking module (see
/// pkg/module/plugin/protocol.go). Like VRCFaceTracking, it calls the
/// module's Update in a loop on its own thread; the host's Update request
/// returns the latest UnifiedTracking.Data at once.
/// </summary>
public sealed class Server(ExtTrackingModule module, JsonLogger logger)
{
    private const byte OpInitialize = (byte)'I', OpUpdate = (byte)'U', OpTeardown = (byte)'T';
    private const byte StatusOk = 0, StatusErr = 1;
    private const byte SupportEye = 1 << 0, SupportExpression = 1 << 1;

    // pkg/tracking.ExpressionCount: the Unified Expressions before Max, in
    // the same order.
    private const int ExpressionCount = 88;
    private const int DataFloats = 2 * 4 + 2 + ExpressionCount + 6;

    private volatile bool _running;
    private bool _initialized;

    public int Serve(Stream input, Stream output)
    {
        var r = new BufferedStream(input);
        var buf = new MemoryStream(1 + DataFloats * 4);
        while (true)
        {
            var op = r.ReadByte();
            if (op < 0) break;
            buf.SetLength(0);
            switch (op)
            {
                case OpInitialize:
                    var avail = r.ReadByte();
                    if (avail < 0) return Exit();
                    try
                    {
                        buf.WriteByte(StatusOk);
                        buf.WriteByte(Initialize((avail & SupportEye) != 0, (avail & SupportExpression) != 0));
                    }
                    catch (Exception e)
                    {
                        WriteError(buf, e);
                    }
                    break;
                case OpUpdate:
                    buf.WriteByte(StatusOk);
                    WriteData(buf);
                    break;
                case OpTeardown:
                    try
                    {
                        Teardown();
                        buf.WriteByte(StatusOk);
                    }
                    catch (Exception e)
                    {
                        WriteError(buf, e);
                    }
                    break;
                default:
                    logger.LogError("Unknown request {op}", op);
                    return Exit(1);
            }
            output.Write(buf.GetBuffer(), 0, (int)buf.Length);
            output.Flush();
        }
        return Exit();
    }

    // The host is gone (or asked us to stop); release the device.
    private int Exit(int code = 0)
    {
        try
        {
            Teardown();
        }
        catch (Exception e)
        {
            logger.LogWarning(e, "Module teardown failed");
        }
        return code;
    }

    private byte Initialize(bool eye, bool expression)
    {
        var (eyeOk, expressionOk) = module.Initialize(eye, expression);
        eyeOk &= eye;
        expressionOk &= expression;
        _initialized = true;
        if (eyeOk || expressionOk)
        {
            module.Status = ModuleState.Active;
            _running = true;
            // Background, so the process can end even if the module blocks
            // in native code.
            new Thread(UpdateLoop) { IsBackground = true, Name = "ModuleUpdate" }.Start();
        }
        return (byte)((eyeOk ? SupportEye : 0) | (expressionOk ? SupportExpression : 0));
    }

    private void UpdateLoop()
    {
        try
        {
            while (_running) module.Update();
        }
        catch (Exception e)
        {
            // VRCFaceTracking's module process dies here too; the host
            // restarts the module.
            logger.LogError(e, "Module update failed");
            Environment.Exit(1);
        }
    }

    private void Teardown()
    {
        _running = false;
        if (!_initialized) return;
        _initialized = false;
        module.Status = ModuleState.Uninitialized;
        module.Teardown();
    }

    private static void WriteError(MemoryStream buf, Exception e)
    {
        buf.SetLength(0);
        var msg = Encoding.UTF8.GetBytes(e.Message);
        buf.WriteByte(StatusErr);
        Span<byte> n = stackalloc byte[4];
        BinaryPrimitives.WriteUInt32LittleEndian(n, (uint)msg.Length);
        buf.Write(n);
        buf.Write(msg);
    }

    // Wire order: per eye (left, right) gaze x, gaze y, pupil diameter,
    // openness; max and min dilation; the shape weights; the head pose.
    private static void WriteData(MemoryStream buf)
    {
        var d = UnifiedTracking.Data;
        var f = new byte[4];
        void Put(float v)
        {
            BinaryPrimitives.WriteSingleLittleEndian(f, v);
            buf.Write(f);
        }
        foreach (var e in new[] { d.Eye.Left, d.Eye.Right })
        {
            Put(e.Gaze.x);
            Put(e.Gaze.y);
            Put(e.PupilDiameter_MM);
            Put(e.Openness);
        }
        Put(d.Eye._maxDilation);
        Put(d.Eye._minDilation);
        for (var i = 0; i < ExpressionCount; i++) Put(d.Shapes[i].Weight);
        Put(d.Head.HeadYaw);
        Put(d.Head.HeadPitch);
        Put(d.Head.HeadRoll);
        Put(d.Head.HeadPosX);
        Put(d.Head.HeadPosY);
        Put(d.Head.HeadPosZ);
    }
}
