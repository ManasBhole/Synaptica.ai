import type { Config } from "tailwindcss";

const config: Config = {
  content: ["./app/**/*.{ts,tsx}", "./components/**/*.{ts,tsx}", "./lib/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        brand: {
          50: "#e0f2fe",
          100: "#bae6fd",
          200: "#7dd3fc",
          400: "#38bdf8",
          500: "#0ea5e9",
          600: "#0284c7"
        },
        accent: {
          400: "#facc15",
          500: "#f59e0b",
          600: "#d97706"
        },
        surface: {
          DEFAULT: "#07111f",
          raised: "#122033"
        }
      },
      boxShadow: {
        floating: "0 18px 45px -22px rgba(14, 165, 233, 0.45)",
        glow: "0 12px 32px rgba(245, 158, 11, 0.35)"
      }
    }
  },
  plugins: []
};

export default config;
