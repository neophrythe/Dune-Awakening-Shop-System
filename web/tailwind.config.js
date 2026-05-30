/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        sand: {
          50: '#faf6ef', 100: '#f2e8d5', 200: '#e6d2ab',
          300: '#d7b67a', 400: '#c97b3c', 500: '#b9692f',
          600: '#9a5326', 700: '#7c4222', 800: '#653720', 900: '#3a2113',
        },
        night: { 800: '#1a1714', 900: '#12100e', 950: '#0b0a08' },
      },
      fontFamily: { display: ['Georgia', 'serif'] },
    },
  },
  plugins: [],
}
