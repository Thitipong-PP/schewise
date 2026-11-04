# Frontend design

Schewise is a database workbench. Its interface should feel quiet, precise, and
ready for repeated use. The login page is the first surface using this
direction.

## Visual language

- Use white and cool neutral surfaces with charcoal text. Reserve deep green for
  primary actions, focus, and the product mark; red indicates errors.
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

- Use `lucide-react` for consistent icons instead of hand-drawn SVG controls. It
  is the only added UI dependency; the project still uses ordinary React
  components and CSS.
- Give icon-only controls accessible names and descriptive hover titles. Keep
  visible keyboard focus, explicit labels, password-manager autocomplete, and
  native form validation. Password visibility is a reversible, labeled action.
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
