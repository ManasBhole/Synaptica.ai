import type { Config } from "tailwindcss";

const config: Config = {
  content: ["./app/**/*.{ts,tsx}", "./components/**/*.{ts,tsx}", "./lib/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        brand: {
          50: "#f5f3ff",
          100: "#ede9fe",
          200: "#ddd6fe",
          400: "#a855f7",
          500: "#9333ea",
          600: "#7e22ce"
        },
        accent: {
          400: "#fb7185",
          500: "#f43f5e",
          600: "#e11d48"
        },
        surface: {
          DEFAULT: "#111827",
          raised: "#1f2937"
        }
      },
      boxShadow: {
        floating: "0 15px 45px -20px rgba(147, 51, 234, 0.45)",
        glow: "0 10px 30px rgba(244, 63, 94, 0.35)"
      }
    }
  },
  plugins: []
};

export default config;
