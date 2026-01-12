/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        dark: {
          900: '#0A0A0F',
          800: '#0D1117',
          700: '#161B22',
          600: '#1A1A2E',
          500: '#21262D',
        },
        electric: {
          blue: '#0066FF',
          cyan: '#00D4FF',
          glow: '#0066FF33',
        },
        verdict: {
          verified: '#00C853',
          'mostly-true': '#64DD17',
          mixed: '#FFD600',
          misleading: '#FF9100',
          false: '#FF1744',
          unverifiable: '#9E9E9E',
          satire: '#AA00FF',
        }
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', 'sans-serif'],
        mono: ['JetBrains Mono', 'monospace'],
      },
      animation: {
        'spin-slow': 'spin 3s linear infinite',
        'pulse-glow': 'pulse-glow 2s ease-in-out infinite',
        'float': 'float 6s ease-in-out infinite',
      },
      keyframes: {
        'pulse-glow': {
          '0%, 100%': { opacity: 0.4 },
          '50%': { opacity: 1 },
        },
        'float': {
          '0%, 100%': { transform: 'translateY(0)' },
          '50%': { transform: 'translateY(-20px)' },
        }
      },
      backgroundImage: {
        'gradient-radial': 'radial-gradient(var(--tw-gradient-stops))',
      }
    },
  },
  plugins: [],
}
