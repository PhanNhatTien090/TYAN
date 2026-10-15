import packageJson from "../../package.json";

const currentYear = new Date().getFullYear();

export const APP_CONFIG = {
  name: "TYAN",
  version: packageJson.version,
  copyright: `© ${currentYear}, TYAN.`,
  meta: {
    title: "TYAN — Security Operations",
    description: "AI-powered security research and assessment console. Built on ARTEX.",
  },
};
