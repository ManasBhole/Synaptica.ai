import type { Config } from "tailwindcss";

const config: Config = {
  content: ["./app/**/*.{ts,tsx}", "./components/**/*.{ts,tsx}", "./lib/**/*.{ts,tsx}"],
  theme: {
    extend: {
      fontFamily: {
        sans: ["Inter", "system-ui", "Segoe UI", "sans-serif"]
      },
      colors: {
        brand: {
          50: "#ecfeff",
          100: "#cffafe",
          200: "#a5f3fc",
          400: "#2dd4bf",
          500: "#0ea5e9",
          600: "#0284c7"
        },
        accent: {
          300: "#fcd34d",
          400: "#fbbf24",
          500: "#f59e0b",
          600: "#d97706"
        },
        surface: {
          subtle: "#f8fafc",
          DEFAULT: "#ffffff",
          raised: "#f1f5f9"
        },
        neutral: {
          50: "#f8fafc",
          100: "#f1f5f9",
          200: "#e2e8f0",
          300: "#cbd5f5",
          600: "#475569",
          700: "#334155",
          900: "#0f172a"
        }
      },
      boxShadow: {
        floating: "0 18px 45px -22px rgba(15, 118, 110, 0.25)",
        glow: "0 18px 34px -20px rgba(245, 158, 11, 0.4)",
        card: "0 20px 40px -24px rgba(15, 23, 42, 0.2)"
      }
    }
  },
  plugins: []
};

export default config;
