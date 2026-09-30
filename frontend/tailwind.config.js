/** @type {import('tailwindcss').Config} */
export default {
  darkMode: 'class',
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        /* ── Surfaces ────────────────────────────────────────────── */
        background: {
          primary: 'var(--bg-primary)',
          secondary: 'var(--bg-secondary)',
          tertiary: 'var(--bg-tertiary)',
          elevated: 'var(--bg-elevated)',
          border: 'var(--bg-border)',
        },
        /* ── Text ────────────────────────────────────────────────── */
        text: {
          primary: 'var(--text-primary)',
          secondary: 'var(--text-secondary)',
          tertiary: 'var(--text-tertiary)',
          muted: 'var(--text-muted)',
        },
        /* ── Accent (semantic) ───────────────────────────────────── */
        accent: {
          primary: 'var(--accent-primary)',
          secondary: 'var(--accent-secondary)',
          success: 'var(--accent-success)',
          danger: 'var(--accent-danger)',
          warning: 'var(--accent-warning)',
          info: 'var(--accent-info)',
        },
        /* ── Core palette (constant) ─────────────────────────────── */
        sky: {
          400: 'var(--sky-400)',
          500: 'var(--sky-500)',
          600: 'var(--sky-600)',
        },
        amber: {
          400: 'var(--amber-400)',
          500: 'var(--amber-500)',
          600: 'var(--amber-600)',
        },
        emerald: {
          500: 'var(--emerald-500)',
        },
        red: {
          500: 'var(--red-500)',
        },
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', '-apple-system', 'BlinkMacSystemFont', 'Segoe UI', 'Roboto', 'sans-serif'],
        mono: ['JetBrains Mono', 'Monaco', 'Courier New', 'monospace'],
      },
      borderRadius: {
        'sm': '4px',
        'md': '6px',
        'lg': '8px',
        'xl': '12px',
      },
      boxShadow: {
        'subtle': 'var(--shadow-subtle)',
        'card': 'var(--shadow-card)',
        'elevated': 'var(--shadow-elevated)',
      },
      animation: {
        'fade-in': 'fade-in 0.2s ease-out',
        'slide-in': 'slide-in 0.3s ease-out',
      },
      keyframes: {
        'fade-in': {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' },
        },
        'slide-in': {
          '0%': { transform: 'translateY(-10px)', opacity: '0' },
          '100%': { transform: 'translateY(0)', opacity: '1' },
        },
      },
    },
  },
  plugins: [],
}
