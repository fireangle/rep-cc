import express, { type Express } from "express";
import cors from "cors";
import pinoHttp from "pino-http";
import path from "path";
import fs from "fs";
import router from "./routes";
import proxyRouter from "./routes/proxy";
import { logger } from "./lib/logger";

const app: Express = express();

app.use(
  pinoHttp({
    logger,
    serializers: {
      req(req) {
        return {
          id: req.id,
          method: req.method,
          url: req.url?.split("?")[0],
        };
      },
      res(res) {
        return {
          statusCode: res.statusCode,
        };
      },
    },
  }),
);
app.use(cors());
app.use(express.json({ limit: "50mb" }));
app.use(express.urlencoded({ extended: true, limit: "50mb" }));

app.use("/api", router);
app.use("/v1", proxyRouter);

const __dirname = path.dirname(new URL(import.meta.url).pathname);
const portalDistPath = path.resolve(__dirname, "../../api-portal/dist/public");
if (fs.existsSync(portalDistPath)) {
  app.use(express.static(portalDistPath));
  app.use((_req, res) => {
    res.sendFile(path.join(portalDistPath, "index.html"));
  });
}

export default app;
