# DESIGN.md — Akashic: The Night Archive

Brand-side design specification. This document describes the *world* Akashic lives in,
not one specific screen. It compounds across every future artifact (about dialog, help,
settings, extensions).

## 1. Objective

Akashic is a local-first, AI-enhanced notepad. It holds a person's private writing —
notes, drafts, code, records of thought — and pairs it with an LLM (Ollama) that runs
entirely on their own machine. The product promise is sovereignty: *your words never
leave this room.*

The interface must therefore feel like a **private instrument**, not a web service.
No SaaS energy. No chat-app energy. It should read as the kind of tool a serious
writer keeps on a desk at night: quiet, precise, slightly old-fashioned in its manners,
modern in its behavior.

Design keywords: **archival, nocturnal, serif, precise, sovereign.**

## 2. Product Context

- Desktop app (Wails: Go backend, native webview frontend). Runs offline by design.
- Primary user: a writer/developer who keeps long-lived documents and wants AI help
  without sending data anywhere.
- The "Akashic Records" mythos (a compendium of all knowledge) is the emotional brand
  anchor. We borrow its *gravity*, not its mysticism — no fantasy UI.
- Constraints: no external font/CDN requests (must work offline). Plain JS, no framework.
  Every existing DOM id/class is load-bearing; the redesign restyles and re-skins, it
  does not rewire.

## 3. Visual Foundations

### Palette

Ink surfaces (cool, blue-undertone near-blacks — never neutral gray like generic IDEs):

```
--ink-0: #0b0e14   window / deepest surface
--ink-1: #10141c   editor surface
--ink-2: #151a24   panels, menus, dialogs
--ink-3: #1c222e   hover, inputs, chips
--ink-4: #262e3d   pressed / active-tint base
--hairline: rgba(206, 214, 232, 0.13)   all borders
```

Paper text (warm, candle-lit — the counterpoint to cool ink):

```
--paper:     #ece6d8   primary text
--paper-dim: #a9a494   secondary text
--paper-faint:#6e6a5e   inactive / disabled
```

Accents — one ember for action, one phosphor for the machine, never both at once:

```
--ember:       #df8a4f   primary accent (hover #eba869, soft rgba(223,138,79,.14))
--phosphor:    #7fd4c1   AI presence / online / success
--gold:        #d4b06a   warning
--brick:       #d96a5f   destructive / error
```

Rules: no purple, no blue-gradient heroes, no `linear-gradient` fills except one
deliberate radial glow behind the AI welcome mark. Contrast floor: paper on ink-1 ≥ 9:1.

### Typography

No web fonts, no CDN. System stacks, chosen for character:

```
--font-serif: Georgia, "Times New Roman", serif       /* voice + editor body */
--font-sans:  "Segoe UI", "Helvetica Neue", sans-serif /* small UI chrome only */
--font-mono:  "Cascadia Code", Consolas, monospace     /* data, code, status */
```

Scale: 11 / 12 / 13 / 14 / 15 / 18 / 24 / 32.

Voice rules:
- Serif is the protagonist. Editor body is serif 15px/1.75. Panel titles, dialog
  headers, menu names, and the AI welcome headline are serif.
- UI chrome (buttons, inputs, chips, status) is small sans, 11–13px, with generous
  letter-spacing; section labels are uppercase small-caps serif.
- Data that looks like data (encodings, zoom %, cursor position, chat timestamps,
  model names, shortcut keys) is mono.

### Shape & Depth

- Radii: 2px (micro), 4px (standard), 6px (large, rare). No pills, no 16px cards.
- Depth comes from hairlines and tonal layering, not shadows. Panels sit flush.
- Shadows exist only for transient chrome (menus, dialogs, toasts) — soft, low-opacity.
- Density: a native desktop cadence. 4px spacing grid; compact chrome, airy editor.

### Texture & Atmosphere

- Whole-app film grain: an inline SVG noise overlay at ~4% opacity, pointer-events off.
- AI welcome state: a sparse dotted "constellation" field (radial-gradient dots),
  anchored by a faint ember radial glow.
- Status dot breathing (opacity pulse) only while checking.

### Motion

- Window open: one staged reveal — chrome fades/rises 8px in three waves (menus →
  tabs/toolbar → editor), 180ms each, 60ms stagger. Nothing else animates on load.
- Menus/dialogs: 140ms fade + 4px rise. Toasts: slide in from the right.
- Micro-interactions: hover states shift tone (never scale or lift); primary buttons
  glow ember on hover; tabs reveal their close glyph; the AI send button warms when
  enabled. `prefers-reduced-motion` honored by dropping non-essential motion.

## 4. Accessibility

- All text meets WCAG AA on its surface (paper ≥ 9:1, paper-dim ≥ 7:1 on ink-2).
- Focus: 1px ember ring + 2px ember-soft halo, visible on every interactive element.
- All icons are inline SVG with `aria-hidden` where the label text already exists;
  icon-only controls carry `title`/`aria-label`.
- Keyboard: full menu/tab/dialog flow preserved; shortcuts shown in menus, mono kbd.
- Fonts: no `font-synthesis` surprises — serif/mono declared with explicit fallbacks.
- `prefers-reduced-motion`: disable staged reveal, breathing, and float animations.

## 5. Voice & Tone

- **The UI speaks like an archivist, not a marketer.** Labels are plain nouns
  ("File", "Chat History" → "Conversations", "Clear All" → "Clear all records").
- No exclamation marks in UI copy. No marketing superlatives. No "AI Assistant"
  sparkle-talk — the machine is named plainly ("Local model").
- The AI welcome is an invitation, not a pitch: "Begin a record" over
  "How can I help you today?".
- Errors are factual and calm: "Ollama is not running. Start it from the status row."

## 6. Implementation Practices

- Single CSS file (`style.css`) is the only stylesheet; tokens live on `:root`.
- Every icon is an inline SVG, 24×24 viewBox, `stroke="currentColor"`,
  `stroke-width="1.5"`, `fill="none"`, sized 14–18px. No icon font, no emoji,
  no unicode glyphs in the UI (×, ↻, ●, → are all replaced by SVG).
- Class names are the contract with main.js — renaming requires touching JS.
- `hidden` utility stays `display:none !important`.
- No framework, no build step beyond vite bundling.

## 7. Anti-Patterns (banned)

- Emoji anywhere in UI chrome, menus, dialogs, or chat.
- Purple/blue gradient heroes or radial "AI glow" outside the one welcome exception.
- ChatGPT-style colored user bubbles; gradient-filled buttons; glassmorphism.
- Rounded-16px shadow card grids; icon+heading+blurb trios.
- Inter/Roboto/system-sans as the visual voice; pill buttons; 47%-YoY stat cards.
- VS Code neutral-gray chrome (the current design is exactly this — it is the
  baseline we are escaping).

## 8. Decision-Making

When in doubt, ask: *would this fit on the desk of the person who wrote the Akashic
Records?* If it looks like a SaaS dashboard, it's wrong. Prefer: serif over sans,
hairline over shadow, warm over neon, quiet over loud, one accent over many.

## 9. Workflow

1. DESIGN.md governs; screen-level decisions are recorded in the Decision Trace.
2. New screens/artifacts start from this spec — never regenerate the palette or
   type voice per-artifact.
3. Any amendment (new accent, new component family) is appended as a trace entry.
