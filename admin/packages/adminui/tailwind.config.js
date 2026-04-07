/** @type {import('tailwindcss').Config} */
module.exports = {
    // 配置 Tailwind 扫描你的 HTML 和 JSX/TSX 文件
    content: [
        "./index.html", // Vite 项目的入口 HTML 文件
        "./src/**/*.{js,jsx,ts,tsx}", // 你的 React 组件文件
    ],
    theme: {
        extend: {
            colors: {
                claw: {
                    50: "#fff8f8",
                    100: "#fff0f0",
                    200: "#ffe0e0",
                    300: "#ffc7c7",
                    400: "#ffa3a3",
                    500: "#f98080",
                    600: "#ea666b",
                    700: "#d9535b",
                    800: "#b9434d",
                    900: "#91343d",
                    950: "#65232a",
                },
            },
        },
    },
    plugins: [],
}
