# Frontend design

Schewise is a database workbench. Its interface should feel quiet, precise, and
ready for repeated use. The login and administration pages use this direction.

## Visual language

- Use a monochrome palette: white surfaces, neutral grays, and charcoal text and
  actions. Keep focus and errors distinguishable with borders, icons, and text.
  A brand color theme has not been selected yet.
- Use the text "Schewise" for the product identity, without a logo icon. The
  final logo will be designed separately.
- Use the bundled Host Grotesk for the product identity and Instrument Sans for
  controls and reading text. Set letter spacing to zero. Keep font sizes fixed
  at each component's intended scale, without viewport-based scaling.
- Prefer unframed layouts, thin dividers, and compact spacing. Avoid marketing
  heroes, decorative illustrations, floating page cards, gradients, and orbs.
- Keep corner radii at 8px or less. Inputs and primary actions use 5px radii and
  a stable 44px height. Focus borders stay close to the control.
- Make the product identity visible without adding feature claims, fabricated
  statistics, connection status, or controls for unavailable capabilities.

## Interaction

- Use `lucide-react` for consistent icons instead of hand-drawn SVG controls.
  The project uses ordinary React components and CSS; a typed API client handles
  server contracts.
- Give icon-only controls accessible names and descriptive hover titles. Keep
  visible keyboard focus, explicit labels, password-manager autocomplete, and
  native form validation. Password visibility is a reversible, labeled action.
- Use one searchable member/user combobox with inline results. Debounce remote
  searches, cancel outdated requests, preserve suggestions while updating, and
  keep stale results unselectable. Searching must not clear role choices or
  remount the form. Support arrow keys, Enter, Escape, and visible selection.
- Native selects retain their behavior with an inset Lucide disclosure icon.
  Inheritance switches require an impact dialog before committing a change.
- Keep pending, disabled, and error states usable without shifting control
  dimensions. Respect reduced motion, including functional loading indicators.
- Keep page and authentication behavior on their existing routes. Do not add
  public registration, recovery links, or provider sign-in without backend
  support.

## Verification

Inspect the rendered desktop and mobile layouts, including narrow 320px screens,
keyboard traversal, password visibility, and failure states. Check for
horizontal overflow and text collisions. Run the source checks and production
build, and audit dependency changes. Provide a local preview URL for design
review.
