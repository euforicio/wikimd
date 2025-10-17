const defaultTheme = require("tailwindcss/defaultTheme");

const withOpacity = (variable) => {
  return ({ opacityValue }) => {
    if (opacityValue === undefined) {
      return `rgb(var(${variable}) / 1)`;
    }
    return `rgb(var(${variable}) / ${opacityValue})`;
  };
};

module.exports = {
  darkMode: "class",
  content: [
    "../internal/server/templates/**/*.gohtml",
    "../static/js/**/*.js",
    "./src/**/*.css",
  ],
  theme: {
    extend: {
      fontFamily: {
        sans: ["Inter", ...defaultTheme.fontFamily.sans],
        mono: ["JetBrains Mono", ...defaultTheme.fontFamily.mono],
      },
      colors: {
        surface: {
          DEFAULT: withOpacity("--surface"),
          subtle: withOpacity("--surface-subtle"),
          accent: withOpacity("--surface-accent"),
          elevated: withOpacity("--surface-elevated"),
          border: withOpacity("--surface-border"),
        },
        brand: {
          emerald: "#00cc6a",
          sky: "#38bdf8",
          amber: "#ffc107",
        },
      },
      boxShadow: {
        card: "0 18px 40px -22px rgba(15, 23, 42, 0.65)",
      },
    },
  },
  plugins: [
    require("@tailwindcss/typography"),
    require("daisyui"),
  ],
  daisyui: {
    themes: [
      {
        wikimd: {
          ...require("daisyui/src/theming/themes")["[data-theme=dark]"],
          "primary": "#00cc6a",
          "secondary": "#34d399",
          "accent": "#ffc107",
          "neutral": "#1c2232",
          "base-100": "#1a1a1a",
          "base-200": "#242424",
          "base-300": "#2d2d2d",
          "info": "#38bdf8",
          "success": "#00cc6a",
          "warning": "#ffc107",
          "error": "#f87171",
        },
      },
      "light",
    ],
  },
};
