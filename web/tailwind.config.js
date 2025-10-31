/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {},
  },
  plugins: [],
  safelist: [
    'px-6', 'py-3', 'rounded-full', 'text-base', 'font-semibold', 'cursor-pointer',
    'transition-all', 'duration-300', 'uppercase', 'tracking-wide',
    'disabled:opacity-60', 'disabled:cursor-not-allowed', 'disabled:transform-none',
    'bg-gradient-to-r', 'from-purple-500', 'to-purple-700', 'text-white',
    'hover:translate-y-[-2px]', 'hover:shadow-lg', 'hover:shadow-purple-500/30',
    'from-pink-400', 'to-red-500', 'hover:shadow-pink-500/30',
    'from-blue-400', 'to-cyan-400', 'hover:shadow-blue-500/30',
    'w-full', 'md:w-auto', 'flex', 'gap-4', 'mb-8', 'flex-wrap', 'justify-center',
    'md:flex-row', 'flex-col'
  ],
}
