# wikimd Themes

This directory contains example themes for wikimd. Customize the appearance using CSS variables without modifying any code!

## Quick Start

### Global Theme (applies to all wikis)
```bash
mkdir -p ~/.wikimd
cp dark.css ~/.wikimd/custom.css
```

### Per-Wiki Theme (specific to one wiki)
```bash
mkdir -p <your-wiki>/.wikimd
cp solarized-dark.css <your-wiki>/.wikimd/custom.css
```

## Available Themes

### Dark (`dark.css`)
Enhanced contrast dark theme with deep blacks and bright text. Perfect for late-night coding sessions.

### Solarized Dark (`solarized-dark.css`)
Classic Solarized Dark color scheme with low contrast and eye-friendly colors.

### Light (`light.css`)
Clean and bright light theme for daytime use.

## CSS Loading Order

Styles are loaded in this priority (later styles override earlier):
1. **Embedded defaults** → Built-in dark theme
2. **Global custom** → `~/.wikimd/custom.css` (your personal theme)
3. **Per-wiki custom** → `<wiki-root>/.wikimd/custom.css` (project-specific)

## Customizable CSS Variables

You can override any of these variables in your `custom.css`:

### Colors - Background
```css
--color-bg-primary: #1a1a1a;
--color-bg-secondary: #242424;
--color-bg-tertiary: #2d2d2d;
--color-bg-code: #000000;
--color-bg-elevated: #333333;
```

### Colors - Text
```css
--color-text-primary: #f1f5f9;
--color-text-secondary: #cbd5e1;
--color-text-muted: #94a3b8;
--color-text-link: #10b981;
--color-text-code: #e2e8f0;
```

### Colors - Borders
```css
--color-border: #3a3a3a;
--color-border-dark: #2d2d2d;
--color-border-light: #475569;
```

### Colors - Accent
```css
--color-accent: #10b981;
--color-accent-hover: #059669;
--color-success: #10b981;
--color-warning: #f59e0b;
--color-error: #ef4444;
```

### Typography
```css
--font-family-base: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
--font-family-mono: "JetBrains Mono", "SF Mono", Monaco, Menlo, monospace;
--font-size-base: 16px;
--font-size-sm: 14px;
--line-height-base: 1.6;
```

### Layout
```css
--sidebar-width: 280px;
--content-max-width: 900px;
--border-radius: 0.5rem;
```

### Effects
```css
--shadow-sm: 0 1px 3px rgba(0, 0, 0, 0.5);
--shadow-md: 0 4px 6px rgba(0, 0, 0, 0.4);
--shadow-lg: 0 10px 25px rgba(0, 0, 0, 0.6);
```

## Creating Your Own Theme

1. Start with one of the example themes
2. Copy it to `custom.css` in your desired location
3. Modify the CSS variables to match your preferences
4. Refresh your browser (Cmd/Ctrl + Shift + R for hard refresh)

### Example: Custom Purple Theme

```css
/* Purple Theme */
:root {
  --color-accent: #a78bfa;
  --color-accent-hover: #8b5cf6;
  --color-text-link: #c084fc;
  --color-sidebar-active: #a78bfa;
}
```

## Live Editing

Custom CSS files are served with `Cache-Control: no-cache`, so you can:
1. Edit your `custom.css` file
2. Save the changes
3. Refresh your browser
4. See the updated theme immediately!

## Tips

- **Start small**: Override just 1-2 variables at first
- **Use browser DevTools**: Inspect elements to see which CSS variables they use
- **Test contrast**: Ensure text is readable on backgrounds
- **Share themes**: Custom themes can be committed with your wiki repo

## Troubleshooting

**Styles not loading?**
- Ensure file is named exactly `custom.css`
- Check location: `~/.wikimd/custom.css` or `<wiki>/.wikimd/custom.css`
- Hard refresh browser (Cmd+Shift+R / Ctrl+Shift+R)

**Per-wiki theme not overriding global?**
- Per-wiki CSS loads AFTER global, so it should win
- Check file exists in correct location

**Want to disable theming?**
- Remove or rename the `custom.css` files
- No configuration needed!
