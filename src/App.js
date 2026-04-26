import React from "react";
import { Routes, Route, Navigate } from "react-router-dom";
import Layout from "./components/Layout";
import Home from "./pages/Home";
import Paper from "./pages/Paper";
import Spec from "./pages/Spec";
import Verify from "./pages/Verify";
import Pricing from "./pages/Pricing";
import Welcome from "./pages/Welcome";
import Account from "./pages/Account";

export default function App() {
  return (
    <Layout>
      <Routes>
        <Route path="/" element={<Home />} />
        <Route path="/paper" element={<Paper />} />
        <Route path="/spec" element={<Spec />} />
        <Route path="/verify" element={<Verify />} />
        <Route path="/pricing" element={<Pricing />} />
        <Route path="/welcome" element={<Welcome />} />
        <Route path="/account" element={<Account />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </Layout>
  );
}
