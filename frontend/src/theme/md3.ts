// Material Design 3 theme for Naive UI.
//
// Colours come from the official material-color-utilities (HCT + DynamicScheme,
// TonalSpot variant, standard contrast); every other value is a token from the
// M3 spec (m3.material.io): shape scale, type scale, state layers, elevation
// and motion easing.
import {
  argbFromHex, Blend, DynamicScheme, Hct, hexFromArgb, MaterialDynamicColors, SchemeTonalSpot, TonalPalette,
} from '@material/material-color-utilities'
import type { GlobalThemeOverrides } from 'naive-ui'

const mdc = new MaterialDynamicColors()

const roles = {
  primary: mdc.primary(),
  onPrimary: mdc.onPrimary(),
  primaryContainer: mdc.primaryContainer(),
  onPrimaryContainer: mdc.onPrimaryContainer(),
  inversePrimary: mdc.inversePrimary(),
  secondary: mdc.secondary(),
  onSecondary: mdc.onSecondary(),
  secondaryContainer: mdc.secondaryContainer(),
  onSecondaryContainer: mdc.onSecondaryContainer(),
  tertiary: mdc.tertiary(),
  onTertiary: mdc.onTertiary(),
  tertiaryContainer: mdc.tertiaryContainer(),
  onTertiaryContainer: mdc.onTertiaryContainer(),
  error: mdc.error(),
  onError: mdc.onError(),
  errorContainer: mdc.errorContainer(),
  onErrorContainer: mdc.onErrorContainer(),
  background: mdc.background(),
  onBackground: mdc.onBackground(),
  surface: mdc.surface(),
  surfaceDim: mdc.surfaceDim(),
  surfaceBright: mdc.surfaceBright(),
  surfaceContainerLowest: mdc.surfaceContainerLowest(),
  surfaceContainerLow: mdc.surfaceContainerLow(),
  surfaceContainer: mdc.surfaceContainer(),
  surfaceContainerHigh: mdc.surfaceContainerHigh(),
  surfaceContainerHighest: mdc.surfaceContainerHighest(),
  surfaceVariant: mdc.surfaceVariant(),
  onSurface: mdc.onSurface(),
  onSurfaceVariant: mdc.onSurfaceVariant(),
  inverseSurface: mdc.inverseSurface(),
  inverseOnSurface: mdc.inverseOnSurface(),
  outline: mdc.outline(),
  outlineVariant: mdc.outlineVariant(),
  shadow: mdc.shadow(),
  scrim: mdc.scrim(),
  surfaceTint: mdc.surfaceTint(),
}

type Role = keyof typeof roles

// Colours the app needs that M3 has no role for. They are harmonized towards
// the seed and given container roles as described in the M3 "custom colors"
// guidance.
const customColors = {
  warning: 0xffffb300,
  success: 0xff2e7d32,
  logWarn: 0xffffff23,
  logError: 0xffdb2543,
  logFatal: 0xffa70023,
}

type Custom = keyof typeof customColors

export type Palette = Record<Role, number> & Record<Custom | `on${Capitalize<Custom>}` | `${Custom}Container` | `on${Capitalize<Custom>}Container`, number>

// State layer opacities (m3.material.io/foundations/interaction/states).
export const stateLayer = { hover: 0.08, focus: 0.1, pressed: 0.1, dragged: 0.16 }
const disabledContent = 0.38
const disabledContainer = 0.12

function capitalize<S extends string>(s: S) {
  return (s[0].toUpperCase() + s.slice(1)) as Capitalize<S>
}

export function generatePalette(seedHex: string, isDark: boolean): Palette {
  const source = argbFromHex(seedHex)
  const scheme: DynamicScheme = new SchemeTonalSpot(Hct.fromInt(source), isDark, 0)
  const out = {} as Record<string, number>
  for (const [name, color] of Object.entries(roles)) out[name] = scheme.getArgb(color)
  for (const [name, argb] of Object.entries(customColors)) {
    const tones = TonalPalette.fromInt(Blend.harmonize(argb, source))
    const n = capitalize(name)
    out[name] = tones.tone(isDark ? 80 : 40)
    out[`on${n}`] = tones.tone(isDark ? 20 : 100)
    out[`${name}Container`] = tones.tone(isDark ? 30 : 90)
    out[`on${n}Container`] = tones.tone(isDark ? 90 : 10)
  }
  return out as Palette
}

const hex = (argb: number) => hexFromArgb(argb)

function rgba(argb: number, alpha: number) {
  return `rgba(${(argb >> 16) & 255}, ${(argb >> 8) & 255}, ${argb & 255}, ${alpha})`
}

// overlay returns base with a state layer of colour `layer` at `opacity` on top.
function overlay(base: number, layer: number, opacity: number) {
  const mix = (shift: number) =>
    Math.round(((base >> shift) & 255) * (1 - opacity) + ((layer >> shift) & 255) * opacity)
  return `rgb(${mix(16)}, ${mix(8)}, ${mix(0)})`
}

function kebab(s: string) {
  return s.replace(/[A-Z]/g, (c) => '-' + c.toLowerCase())
}

// Elevation levels 1–3 (m3.material.io/styles/elevation/tokens).
function elevation(shadow: number) {
  const key = rgba(shadow, 0.3)
  const ambient = rgba(shadow, 0.15)
  return [
    `0 1px 2px 0 ${key}, 0 1px 3px 1px ${ambient}`,
    `0 1px 2px 0 ${key}, 0 2px 6px 2px ${ambient}`,
    `0 1px 3px 0 ${key}, 0 4px 8px 3px ${ambient}`,
  ]
}

// Shape scale (m3.material.io/styles/shape/corner-radius-scale).
export const shape = { extraSmall: '4px', small: '8px', medium: '12px', large: '16px', extraLarge: '28px', full: '9999px' }

// Easing tokens (m3.material.io/styles/motion/easing-and-duration/tokens-specs).
const easing = {
  standard: 'cubic-bezier(0.2, 0, 0, 1)',
  emphasizedDecelerate: 'cubic-bezier(0.05, 0.7, 0.1, 1)',
  emphasizedAccelerate: 'cubic-bezier(0.3, 0, 0.8, 0.15)',
}

/** cssVariables returns the palette as `--md-sys-color-*` custom properties. */
export function cssVariables(p: Palette): Record<string, string> {
  const vars: Record<string, string> = {}
  for (const [name, argb] of Object.entries(p)) vars[`--md-sys-color-${kebab(name)}`] = hex(argb)
  vars['--md-sys-color-on-surface-hover'] = rgba(p.onSurface, stateLayer.hover)
  vars['--md-sys-color-on-surface-pressed'] = rgba(p.onSurface, stateLayer.pressed)
  return vars
}

export function themeOverrides(p: Palette): GlobalThemeOverrides {
  const [shadow1, shadow2, shadow3] = elevation(p.shadow)
  // Filled containers get a state layer of their on-colour.
  const family = (base: number, on: number) => ({
    base: hex(base),
    hover: overlay(base, on, stateLayer.hover),
    pressed: overlay(base, on, stateLayer.pressed),
  })
  const primary = family(p.primary, p.onPrimary)
  const error = family(p.error, p.onError)
  const warning = family(p.warning, p.onWarning)
  const success = family(p.success, p.onSuccess)
  const disabledText = rgba(p.onSurface, disabledContent)
  const disabledFill = rgba(p.onSurface, disabledContainer)
  const hoverLayer = rgba(p.onSurface, stateLayer.hover)
  const pressedLayer = rgba(p.onSurface, stateLayer.pressed)

  return {
    common: {
      baseColor: hex(p.surface),
      primaryColor: primary.base,
      primaryColorHover: primary.hover,
      primaryColorPressed: primary.pressed,
      primaryColorSuppl: primary.hover,
      infoColor: primary.base,
      infoColorHover: primary.hover,
      infoColorPressed: primary.pressed,
      infoColorSuppl: primary.hover,
      successColor: success.base,
      successColorHover: success.hover,
      successColorPressed: success.pressed,
      successColorSuppl: success.hover,
      warningColor: warning.base,
      warningColorHover: warning.hover,
      warningColorPressed: warning.pressed,
      warningColorSuppl: warning.hover,
      errorColor: error.base,
      errorColorHover: error.hover,
      errorColorPressed: error.pressed,
      errorColorSuppl: error.hover,
      textColorBase: hex(p.onSurface),
      textColor1: hex(p.onSurface),
      textColor2: hex(p.onSurface),
      textColor3: hex(p.onSurfaceVariant),
      textColorDisabled: disabledText,
      placeholderColor: hex(p.onSurfaceVariant),
      placeholderColorDisabled: disabledText,
      iconColor: hex(p.onSurfaceVariant),
      iconColorHover: hex(p.onSurface),
      iconColorPressed: hex(p.onSurface),
      iconColorDisabled: disabledText,
      dividerColor: hex(p.outlineVariant),
      borderColor: hex(p.outline),
      closeIconColor: hex(p.onSurfaceVariant),
      closeIconColorHover: hex(p.onSurfaceVariant),
      closeIconColorPressed: hex(p.onSurfaceVariant),
      closeColorHover: rgba(p.onSurfaceVariant, stateLayer.hover),
      closeColorPressed: rgba(p.onSurfaceVariant, stateLayer.pressed),
      clearColor: hex(p.onSurfaceVariant),
      clearColorHover: hex(p.onSurface),
      clearColorPressed: hex(p.onSurface),
      scrollbarColor: rgba(p.onSurfaceVariant, 0.38),
      scrollbarColorHover: rgba(p.onSurfaceVariant, 0.6),
      progressRailColor: hex(p.secondaryContainer),
      railColor: hex(p.surfaceContainerHighest),
      popoverColor: hex(p.surfaceContainer),
      tableColor: hex(p.surface),
      cardColor: hex(p.surfaceContainerLow),
      modalColor: hex(p.surfaceContainerHigh),
      bodyColor: hex(p.surface),
      tagColor: hex(p.surfaceContainerLow),
      avatarColor: hex(p.secondaryContainer),
      invertedColor: hex(p.inverseSurface),
      inputColor: 'transparent',
      inputColorDisabled: 'transparent',
      codeColor: hex(p.surfaceContainerHighest),
      tabColor: hex(p.surfaceContainer),
      actionColor: hex(p.surfaceContainer),
      tableHeaderColor: hex(p.surfaceContainer),
      hoverColor: hoverLayer,
      tableColorHover: hoverLayer,
      tableColorStriped: hex(p.surfaceContainerLow),
      pressedColor: pressedLayer,
      buttonColor2: hex(p.secondaryContainer),
      buttonColor2Hover: overlay(p.secondaryContainer, p.onSecondaryContainer, stateLayer.hover),
      buttonColor2Pressed: overlay(p.secondaryContainer, p.onSecondaryContainer, stateLayer.pressed),
      boxShadow1: shadow1,
      boxShadow2: shadow2,
      boxShadow3: shadow3,
      cubicBezierEaseInOut: easing.standard,
      cubicBezierEaseOut: easing.emphasizedDecelerate,
      cubicBezierEaseIn: easing.emphasizedAccelerate,
      borderRadius: shape.extraSmall,
      borderRadiusSmall: shape.extraSmall,
      // body-medium
      fontSize: '14px',
      fontSizeMedium: '14px',
      lineHeight: String(20 / 14),
      fontWeightStrong: '500',
    },
    Button: {
      // Common buttons: 40dp high, full corner radius, label-large text.
      heightMedium: '40px',
      paddingMedium: '0 24px',
      borderRadiusTiny: shape.full,
      borderRadiusSmall: shape.full,
      borderRadiusMedium: shape.full,
      borderRadiusLarge: shape.full,
      fontSizeMedium: '14px',
      fontWeight: '400', // label-large; see style.css for why not 500
      // Outlined button.
      border: `1px solid ${hex(p.outline)}`,
      borderHover: `1px solid ${hex(p.outline)}`,
      borderPressed: `1px solid ${hex(p.outline)}`,
      borderFocus: `1px solid ${hex(p.outline)}`,
      borderDisabled: `1px solid ${disabledFill}`,
      textColor: primary.base,
      textColorHover: primary.base,
      textColorPressed: primary.base,
      textColorFocus: primary.base,
      textColorDisabled: disabledText,
      color: 'transparent',
      colorHover: rgba(p.primary, stateLayer.hover),
      colorPressed: rgba(p.primary, stateLayer.pressed),
      colorFocus: rgba(p.primary, stateLayer.focus),
      colorDisabled: 'transparent',
      // Filled button.
      textColorPrimary: hex(p.onPrimary),
      textColorHoverPrimary: hex(p.onPrimary),
      textColorPressedPrimary: hex(p.onPrimary),
      textColorFocusPrimary: hex(p.onPrimary),
      textColorDisabledPrimary: disabledText,
      colorDisabledPrimary: disabledFill,
      textColorError: hex(p.onError),
      textColorHoverError: hex(p.onError),
      textColorPressedError: hex(p.onError),
      textColorFocusError: hex(p.onError),
      // Text button.
      textColorText: primary.base,
      textColorTextHover: primary.base,
      textColorTextPressed: primary.base,
      textColorTextFocus: primary.base,
      textColorTextDisabled: disabledText,
      colorQuaternaryHover: rgba(p.onSurfaceVariant, stateLayer.hover),
      colorQuaternaryPressed: rgba(p.onSurfaceVariant, stateLayer.pressed),
      // Filled tonal button.
      colorSecondary: hex(p.secondaryContainer),
      colorSecondaryHover: overlay(p.secondaryContainer, p.onSecondaryContainer, stateLayer.hover),
      colorSecondaryPressed: overlay(p.secondaryContainer, p.onSecondaryContainer, stateLayer.pressed),
      colorTertiary: 'transparent',
      colorTertiaryHover: rgba(p.onSurfaceVariant, stateLayer.hover),
      colorTertiaryPressed: rgba(p.onSurfaceVariant, stateLayer.pressed),
      rippleDuration: '0s',
    },
    Card: {
      // Elevated card: surface-container-low, medium corners, no outline.
      borderRadius: shape.medium,
      color: hex(p.surfaceContainerLow),
      colorModal: hex(p.surfaceContainerLow),
      colorEmbedded: hex(p.surfaceContainerHighest),
      borderColor: 'transparent',
      titleFontWeight: '400',
      titleTextColor: hex(p.onSurface),
      textColor: hex(p.onSurfaceVariant),
    },
    Dialog: {
      color: hex(p.surfaceContainerHigh),
      borderRadius: shape.extraLarge,
      padding: '24px',
      titleFontSize: '24px', // headline-small
      titleFontWeight: '400',
      titleTextColor: hex(p.onSurface),
      textColor: hex(p.onSurfaceVariant),
      iconColor: hex(p.secondary),
      iconColorInfo: hex(p.secondary),
      iconColorWarning: hex(p.secondary),
      iconColorError: hex(p.error),
      iconColorSuccess: hex(p.secondary),
    },
    Popover: {
      color: hex(p.surfaceContainer),
      textColor: hex(p.onSurface),
      borderRadius: shape.extraSmall,
      boxShadow: shadow2,
    },
    Tooltip: {
      // Plain tooltip.
      color: hex(p.inverseSurface),
      textColor: hex(p.inverseOnSurface),
      borderRadius: shape.extraSmall,
      padding: '4px 8px',
      boxShadow: 'none',
    },
    InternalSelectMenu: {
      color: hex(p.surfaceContainer),
      borderRadius: shape.extraSmall,
      optionHeightMedium: '48px',
      optionTextColor: hex(p.onSurface),
      optionTextColorActive: hex(p.onSurface),
      optionTextColorPressed: hex(p.onSurface),
      optionColorPending: hoverLayer,
      optionColorActive: hex(p.secondaryContainer),
      optionColorActivePending: overlay(p.secondaryContainer, p.onSecondaryContainer, stateLayer.hover),
      optionCheckColor: hex(p.onSecondaryContainer),
      optionPaddingMedium: '0 12px',
    },
    Input: {
      // Outlined text field.
      borderRadius: shape.extraSmall,
      color: 'transparent',
      colorFocus: 'transparent',
      colorDisabled: 'transparent',
      border: `1px solid ${hex(p.outline)}`,
      borderHover: `1px solid ${hex(p.onSurface)}`,
      borderFocus: `2px solid ${hex(p.primary)}`,
      borderDisabled: `1px solid ${disabledFill}`,
      boxShadowFocus: 'none',
      textColor: hex(p.onSurface),
      caretColor: hex(p.primary),
      placeholderColor: hex(p.onSurfaceVariant),
      heightMedium: '40px',
      paddingMedium: '0 16px',
    },
    InternalSelection: {
      borderRadius: shape.extraSmall,
      color: 'transparent',
      colorActive: 'transparent',
      colorDisabled: 'transparent',
      border: `1px solid ${hex(p.outline)}`,
      borderHover: `1px solid ${hex(p.onSurface)}`,
      borderActive: `2px solid ${hex(p.primary)}`,
      borderFocus: `2px solid ${hex(p.primary)}`,
      boxShadowActive: 'none',
      boxShadowFocus: 'none',
      textColor: hex(p.onSurface),
      placeholderColor: hex(p.onSurfaceVariant),
      arrowColor: hex(p.onSurfaceVariant),
      heightMedium: '40px',
      paddingSingle: '0 40px 0 16px',
    },
    Switch: {
      // 52×32 track, 24dp handle when on / 16dp when off (handled in style.css).
      railWidthMedium: '52px',
      railHeightMedium: '32px',
      railBorderRadiusMedium: shape.full,
      buttonWidthMedium: '24px',
      buttonHeightMedium: '24px',
      buttonWidthPressedMedium: '24px',
      buttonBorderRadiusMedium: shape.full,
      railColor: hex(p.surfaceContainerHighest),
      railColorActive: hex(p.primary),
      buttonColor: hex(p.outline),
      buttonBoxShadow: 'none',
      boxShadowFocus: `0 0 0 8px ${rgba(p.onSurface, stateLayer.focus)}`,
      opacityDisabled: '0.38',
    },
    Checkbox: {
      sizeMedium: '18px',
      borderRadius: '2px',
      color: 'transparent',
      colorChecked: hex(p.primary),
      checkMarkColor: hex(p.onPrimary),
      border: `2px solid ${hex(p.onSurfaceVariant)}`,
      borderChecked: `2px solid ${hex(p.primary)}`,
      borderFocus: `2px solid ${hex(p.onSurface)}`,
      boxShadowFocus: `0 0 0 11px ${rgba(p.onSurface, stateLayer.focus)}`,
      textColor: hex(p.onSurface),
    },
    Slider: {
      railColor: hex(p.secondaryContainer),
      railColorHover: hex(p.secondaryContainer),
      fillColor: hex(p.primary),
      fillColorHover: hex(p.primary),
      handleColor: hex(p.primary),
      handleBoxShadow: 'none',
      handleBoxShadowHover: `0 0 0 10px ${rgba(p.primary, stateLayer.hover)}`,
      handleBoxShadowActive: `0 0 0 10px ${rgba(p.primary, stateLayer.pressed)}`,
      handleBoxShadowFocus: `0 0 0 10px ${rgba(p.primary, stateLayer.focus)}`,
      indicatorColor: hex(p.inverseSurface),
      indicatorTextColor: hex(p.inverseOnSurface),
      indicatorBorderRadius: shape.full,
      railHeight: '4px',
      handleSize: '20px',
    },
    Tag: {
      // Chips: small corners, outline.
      borderRadius: shape.small,
      heightMedium: '32px',
      color: 'transparent',
      border: `1px solid ${hex(p.outlineVariant)}`,
      textColor: hex(p.onSurfaceVariant),
    },
    FloatButton: {
      // Lowered FAB: primary-container, large corners, elevation 1 (2 on
      // hover). The default level 3 shadow looks too heavy on this flat UI.
      colorPrimary: hex(p.primaryContainer),
      colorPrimaryHover: overlay(p.primaryContainer, p.onPrimaryContainer, stateLayer.hover),
      colorPrimaryPressed: overlay(p.primaryContainer, p.onPrimaryContainer, stateLayer.pressed),
      textColorPrimary: hex(p.onPrimaryContainer),
      borderRadiusSquare: shape.large,
      boxShadow: shadow1,
      boxShadowHover: shadow2,
      boxShadowPressed: shadow1,
    },
    Collapse: {
      dividerColor: hex(p.outlineVariant),
      titleTextColor: hex(p.onSurface),
      titleFontSize: '16px', // title-medium
      titleFontWeight: '500',
      textColor: hex(p.onSurfaceVariant),
      arrowColor: hex(p.onSurfaceVariant),
    },
    Alert: {
      borderRadius: shape.medium,
    },
    Empty: {
      textColor: hex(p.onSurfaceVariant),
      iconColor: hex(p.onSurfaceVariant),
    },
    Message: {
      // Snackbar.
      color: hex(p.inverseSurface),
      colorInfo: hex(p.inverseSurface),
      colorSuccess: hex(p.inverseSurface),
      colorWarning: hex(p.inverseSurface),
      colorError: hex(p.inverseSurface),
      colorLoading: hex(p.inverseSurface),
      textColor: hex(p.inverseOnSurface),
      textColorInfo: hex(p.inverseOnSurface),
      textColorSuccess: hex(p.inverseOnSurface),
      textColorWarning: hex(p.inverseOnSurface),
      textColorError: hex(p.inverseOnSurface),
      textColorLoading: hex(p.inverseOnSurface),
      iconColorInfo: hex(p.inversePrimary),
      iconColorSuccess: hex(p.inversePrimary),
      iconColorWarning: hex(p.inversePrimary),
      iconColorError: hex(p.inversePrimary),
      borderRadius: shape.extraSmall,
      boxShadow: shadow3,
    },
    Menu: {
      // Navigation drawer (compacted for this small window): 48px items, full-radius secondary-container indicator.
      color: 'transparent',
      itemHeight: '48px',
      borderRadius: shape.full,
      fontSize: '14px',
      itemTextColor: hex(p.onSurfaceVariant),
      itemTextColorHover: hex(p.onSurface),
      itemTextColorActive: hex(p.onSecondaryContainer),
      itemTextColorActiveHover: hex(p.onSecondaryContainer),
      itemIconColor: hex(p.onSurfaceVariant),
      itemIconColorHover: hex(p.onSurface),
      itemIconColorActive: hex(p.onSecondaryContainer),
      itemIconColorActiveHover: hex(p.onSecondaryContainer),
      itemIconColorCollapsed: hex(p.onSurfaceVariant),
      itemColorHover: hoverLayer,
      itemColorActive: hex(p.secondaryContainer),
      itemColorActiveHover: overlay(p.secondaryContainer, p.onSecondaryContainer, stateLayer.hover),
      itemColorActiveCollapsed: hex(p.secondaryContainer),
    },
    Layout: {
      // The window chrome (title bar + drawer) is surface-container; the
      // content pane draws its own surface background.
      color: 'transparent',
      siderColor: 'transparent',
      siderBorderColor: 'transparent',
      headerColor: hex(p.surface),
      siderToggleButtonColor: hex(p.surfaceContainerHigh),
      siderToggleButtonBorder: `1px solid ${hex(p.outlineVariant)}`,
      siderToggleButtonIconColor: hex(p.onSurfaceVariant),
    },
    Scrollbar: {
      color: rgba(p.onSurfaceVariant, 0.38),
      colorHover: rgba(p.onSurfaceVariant, 0.6),
    },
  }
}
