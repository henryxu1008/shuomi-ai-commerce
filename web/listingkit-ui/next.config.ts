import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  allowedDevOrigins: ["127.0.0.1", "localhost"],
  output: "standalone",
  logging:{incomingRequests:{ignore:[/^\/workbench\/stores\/(?:shein|tiktokshop)\/callback(?:\?|$)/]}},
  async headers(){return ["shein","tiktokshop"].map(platform=>({source:`/workbench/stores/${platform}/callback`,headers:[{key:"Referrer-Policy",value:"no-referrer"},{key:"Cache-Control",value:"private, no-store"},{key:"X-Content-Type-Options",value:"nosniff"}]}));},
  images: {
    remotePatterns: [
      {
        protocol: "https",
        hostname: "cdn.sdspod.com",
      },
      {
        protocol: "http",
        hostname: "cdn.sdspod.com",
      },
      {
        protocol: "http",
        hostname: "e.sdspod.com",
      },
      {
        protocol: "https",
        hostname: "e.sdspod.com",
      },
      {
        protocol: "https",
        hostname: "static-photo-center-prov.oss-cn-hangzhou.aliyuncs.com",
      },
    ],
  },
};

export default nextConfig;
