import React from "react";
import { createRoot } from "react-dom/client";
import "../app/globals.css";
import { SetupApp } from "../components/setup/setup-app";

createRoot(document.getElementById("root")!).render(<React.StrictMode><SetupApp /></React.StrictMode>);
