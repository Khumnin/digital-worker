# TigerSoft Branding CI Guidelines

> Source: `guide/CI Toolkit/CI Toolkit.pdf` (40 pages, 2025)
> All UI design and frontend implementation **MUST** align with these guidelines.

---

## Brand DNA

- **Agile** — fast, flexible, adaptive
- **Empowering** — support users to work better
- **Seamless** — system + people + service as one

---

## Logo

| Variant | Usage | File Location |
|---|---|---|
| **Primary Logo** (horizontal) | Default for all media | `guide/Logo Tigersoft 5/` |
| **Vertical Logo** | When horizontal space is limited | `guide/Logo Tigersoft 5/` |
| **Logomark** (icon only) | Favicon, badge, small icons | `guide/Logo Tigersoft 5/` |

### Logo Rules
- Minimum safe space: **2X** around the logo (X = logo height unit)
- Use **original files only** (SVG, PNG) — never recreate
- **DO NOT:** swap logomark/logotype positions, add effects/shadows, rotate, stretch, change colors/fonts, reduce opacity, place on busy backgrounds, or use wrong sizes

---

## Color Palette

### Primary Colors (85% of design)

| Name | HEX | RGB | CMYK | Pantone | Usage |
|---|---|---|---|---|---|
| **Vivid Red** | `#F4001A` | 244, 0, 26 | 0, 100, 100, 0 | 2347 C | CTA buttons, accents, brand highlights (20%) |
| **White** | `#FFFFFF` | 255, 255, 255 | 0, 0, 0, 0 | White 000 C | Backgrounds, cards, content areas (45%) |
| **Oxford Blue** | `#0B1F3A` | 11, 31, 58 | 96, 83, 47, 57 | 289 C | Headings, text, dark backgrounds (20%) |

### Secondary Colors (15% of design)

| Name | HEX | RGB | CMYK | Pantone | Usage |
|---|---|---|---|---|---|
| **Quick Silver** | `#A3A3A3` | 163, 163, 163 | 38, 31, 32, 0 | 422 C | Borders, dividers, disabled states (5%) |
| **Serene** | `#DBE1E1` | 219, 225, 225 | 13, 7, 9, 0 | 7541 C | Light backgrounds, cards, subtle UI (5%) |
| **UFO Green** | `#34D186` | 52, 209, 134 | 66, 0, 67, 0 | 7479 C | Success states, activation, positive actions (5%) |

### Gradient Combinations (allowed)
- White → Vivid Red
- Vivid Red → Oxford Blue
- Oxford Blue → Quick Silver
- Quick Silver → Serene
- Serene → White

---

## Typography

### English: **Plus Jakarta Sans** (Google Fonts)
| Level | Weight | Usage |
|---|---|---|
| Heading | Medium (500) | Page titles, section headers |
| Subheading | Medium (500) | Card titles, sub-sections |
| Paragraph | Light (300) | Body text, descriptions |

### Thai: **FC Vision** (custom font)
| Level | Weight | Usage |
|---|---|---|
| Heading | Medium | Page titles, section headers |
| Subheading | Medium | Card titles, sub-sections |
| Paragraph | Light | Body text, descriptions |

### Typography Rules
- Use the **same font pair consistently** across all platforms
- Maintain clear **visual hierarchy** between Heading, Subheading, and Paragraph
- Font files located at: `guide/CI Toolkit/Font/TH/FC Vision/`

---

## Design Elements

### Key Design Techniques
| Technique | Description |
|---|---|
| **Soft Edges** | Rounded corners on cards, buttons, and containers |
| **Grid System Layout** | Consistent grid-based alignment |
| **Layers** | Overlapping elements to create depth |
| **Clipping Mask** | Rounded/shaped image containers |
| **Glassmorphism** | Frosted glass effect (gaussian blur) with graphic patterns |
| **Graphic Element & Pattern** | Grid + Dot + Connector patterns from brand concept |

### Graphic Elements
- **Growth** — upward curve lines representing progress
- **Dot** — data points in a network
- **Grid** — structure and systematic organization
- **Connector** — lines linking topics, framing content, establishing hierarchy

### Graphic Icons Style
- Simple **outline treatment** matching Grid and Connector elements
- Combination of **sharp and softened corners** — firm yet approachable
- Icon categories: User & Security, Analytics & Data, etc.
- Icons located at: `guide/CI Toolkit/File Assets/Graphic icon/`

---

## UI Design Principles (for web/app)

1. **White-dominant** — 45% white space for clean, professional feel
2. **Oxford Blue for text** — primary text color, never pure black
3. **Vivid Red for CTAs** — primary action buttons and brand accents
4. **UFO Green for success** — confirmations, positive states
5. **Soft edges everywhere** — rounded corners (8px–16px for cards, 4px–8px for inputs)
6. **Grid-based layouts** — consistent spacing and alignment
7. **Glassmorphism sparingly** — for featured sections or hero areas only
8. **Plus Jakarta Sans** — load via Google Fonts for English text
9. **No pure black (#000)** — use Oxford Blue (#0B1F3A) instead
10. **Accessibility** — ensure sufficient contrast ratios (WCAG 2.1 AA)
