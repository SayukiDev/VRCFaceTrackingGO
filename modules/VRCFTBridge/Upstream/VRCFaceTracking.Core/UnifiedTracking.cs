// Trimmed from VRCFaceTracking.Core/UnifiedTracking.cs (Apache-2.0): only
// the members a tracking module writes to. The output parameters, the
// mutator and the update events belong to the VRCFaceTracking application,
// which VRCFaceTrackingGO replaces.
using VRCFaceTracking.Core.Params.Data;
using VRCFaceTracking.Core.Types;

namespace VRCFaceTracking;

/// <summary>
/// Class that contains all relevant data
/// </summary>
public class UnifiedTracking
{
    /// <summary>
    /// Eye image data sent from the loaded eye module.
    /// </summary>
    public static Image EyeImageData = new Image();

    /// <summary>
    /// Lip / Expression image data sent from the loaded expressions module.
    /// </summary>
    public static Image LipImageData = new Image();

    /// <summary>
    /// Latest Expression Data accessible and sent by all VRCFaceTracking modules.
    /// </summary>
    public static UnifiedTrackingData Data = new();
}
