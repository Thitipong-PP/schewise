<!-- SPDX-License-Identifier: Apache-2.0 -->

# ADR 0005: inheritance boundaries and centralized revocation

- Status: accepted
- Date: 2026-11-05
- Extends: ADRs 0003 and 0004

## Context

A subtree manager cannot revoke grants originating at an ancestor. They need an
explicit, reviewable way to stop those grants at their own boundary while
preserving direct access and installation administration.

## Decision

Add `units.inherit_access`, defaulting to true. Collect assignments upward from
the resource and stop after the first false unit. Preserve the built-in root
Administrator bypass; custom clones remain subject to boundaries. Keep physical
ancestry separate from the effective assignment chain.

Authorize changes with `unit:manage_members`, reject root changes and
self-lockout, and reuse the shared mutation lock and atomic audit protocol.
Preview before/after roles, permissions, users, and connections with the
existing projected reader. Hide unreadable grant origins and blocking units
independently. Migration 00003 is additive; downgrade refuses to silently
broaden access.

Centralize revocation in `CanRevoke`, including delegation authority, the
actor's unit subtree, held permissions, and last-root-Administrator protection.
Permit self-revocation only when those checks pass. Retain database guards as
defense in depth.

## Consequences

Inheritance changes can remove access and require a real impact confirmation.
Previews remain descriptive; writes recheck live state. Membership and tree
visibility follow the same engine. The UI uses a debounced member/user combobox,
keeps role selection stable during searches, and disables stale suggestions. No
new target connections, identity features, or external services are added.
