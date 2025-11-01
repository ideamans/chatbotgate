# Multi OAuth2 Proxy - Design System

This directory contains the frontend design system and assets for the Multi OAuth2 Proxy authentication pages.

## Tech Stack

- **Vite** - Fast build tool and dev server
- **Tailwind CSS 4** - Utility-first CSS framework
- **TypeScript** - Type-safe JavaScript

## Getting Started

### Install Dependencies

```bash
yarn install
# or
npm install
```

### Development

Run the development server with hot reload:

```bash
yarn dev
# or
npm run dev
```

This will start a dev server at `http://localhost:3000` showing the design system catalog.

### Build for Production

Build the CSS and assets for embedding in Go:

```bash
yarn build
# or
npm run build
```

This will generate:
- `dist/styles.css` - Compiled CSS (used by Go via embed)
- `dist/assets/*` - Optimized images and other assets

## Design System

### Color System

The design system includes:
- Primary colors (Primary, Secondary)
- Status colors (Success, Warning, Error)
- Background colors (Base, Elevated, Muted)
- Text colors (Primary, Secondary, Muted)
- Border colors

All colors automatically adapt to light/dark mode.

### Components

Available components:
- **Buttons**: Primary, Secondary, Ghost variants
- **Forms**: Input fields, labels, form groups
- **Cards**: Elevated containers
- **Alerts**: Success, Warning, Error states
- **Auth Pages**: Pre-designed login/logout pages

### Theme Support

- Auto-detects system preference (light/dark)
- Manual theme toggle
- Smooth transitions between themes

## File Structure

```
web/
├── src/
│   ├── styles/
│   │   └── main.css          # Main stylesheet with Tailwind CSS 4
│   └── index.html             # Design system catalog
├── public/
│   └── images/                # Static images
├── dist/                      # Build output (committed to git)
│   ├── styles.css            # Generated CSS for Go embed
│   └── assets/               # Optimized assets
├── package.json
├── tsconfig.json
├── vite.config.ts
└── README.md
```

## Integration with Go

The built CSS is embedded in the Go server:

1. Build the assets: `yarn build`
2. Go server reads `web/dist/styles.css` via `//go:embed`
3. CSS is served inline in HTML templates

## Customization

### Adding New Colors

Edit `src/styles/main.css`:

```css
@theme {
  --color-your-color: #hexcode;
}
```

### Adding New Components

Add component styles in `src/styles/main.css` and examples in `src/index.html`.

## Browser Support

- Modern browsers with CSS custom properties support
- Chrome/Edge 88+
- Firefox 78+
- Safari 14+
