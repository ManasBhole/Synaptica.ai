import type { Config } from "tailwindcss";

const config: Config = {
  content: ["./app/**/*.{ts,tsx}", "./components/**/*.{ts,tsx}", "./lib/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        primary: {
          50: "#eff6ff",
          100: "#dbeafe",
          200: "#bfdbfe",
          500: "#2563eb",
          600: "#1d4ed8"
        },
        accent: {
          400: "#38bdf8",
          500: "#0ea5e9"
        },
        slate: {
          950: "#020617"
        }
      },
      boxShadow: {
        floating: "0 20px 45px -25px rgba(15, 118, 110, 0.45)"
      }
    }
  },
  plugins: []
};

export default config;
