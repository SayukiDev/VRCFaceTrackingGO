// A VRCFaceTracking module written exactly as one for VRCFaceTracking
// would be: it derives from ExtTrackingModule and writes into
// UnifiedTracking.Data from Update, which the host calls in a loop. It
// needs no hardware: it opens the jaw, smiles, blinks and looks around.
using Microsoft.Extensions.Logging;
using VRCFaceTracking;
using VRCFaceTracking.Core.Params.Expressions;

namespace DemoModule;

public class DemoModule : ExtTrackingModule
{
    private readonly DateTime _start = DateTime.Now;
    private bool _eye, _expression;

    public override (bool SupportsEye, bool SupportsExpression) Supported => (true, true);

    public override (bool eyeSuccess, bool expressionSuccess) Initialize(bool eyeAvailable, bool expressionAvailable)
    {
        (_eye, _expression) = (eyeAvailable, expressionAvailable);
        ModuleInformation.Name = "VRCFT Demo";
        Logger.LogInformation("Demo module initialized (eye: {eye}, expression: {expression})", _eye, _expression);
        return (_eye, _expression);
    }

    public override void Update()
    {
        var t = (DateTime.Now - _start).TotalSeconds;
        var data = UnifiedTracking.Data;
        if (_expression)
        {
            data.Shapes[(int)UnifiedExpressions.JawOpen].Weight = (float)(0.4 + 0.4 * Math.Sin(t * 2));
            var smile = (float)(0.5 + 0.5 * Math.Sin(t * 0.5));
            data.Shapes[(int)UnifiedExpressions.MouthCornerPullLeft].Weight = smile;
            data.Shapes[(int)UnifiedExpressions.MouthCornerPullRight].Weight = smile;
            data.Head.HeadYaw = (float)(0.2 * Math.Sin(t * 0.7));
        }
        if (_eye)
        {
            var openness = t % 4 < 0.15 ? 0f : 1f;
            var gaze = new VRCFaceTracking.Core.Types.Vector2((float)(0.3 * Math.Cos(t)), (float)(0.2 * Math.Sin(t)));
            data.Eye.Left.Openness = data.Eye.Right.Openness = openness;
            data.Eye.Left.Gaze = data.Eye.Right.Gaze = gaze;
            data.Eye.Left.PupilDiameter_MM = data.Eye.Right.PupilDiameter_MM = 4;
            data.Eye._minDilation = 2;
            data.Eye._maxDilation = 7;
        }
        // VRCFaceTracking modules pace themselves.
        Thread.Sleep(10);
    }

    public override void Teardown() => Logger.LogInformation("Demo module stopped");
}
