import React from "react";
import { Routes, Route, Navigate } from "react-router-dom";
import Layout from "./components/Layout";
import Home from "./pages/Home";
import Paper from "./pages/Paper";
import Spec from "./pages/Spec";
import Welcome from "./pages/Welcome";
import Account from "./pages/Account";
import VerifyMoved from "./pages/VerifyMoved";
import Agent from "./pages/Agent";
import Contact from "./pages/Contact";

export default function App() {
  return (
    <Layout>
      <Routes>
        <Route path="/" element={<Home />} />
        <Route path="/paper" element={<Paper />} />
        <Route path="/spec" element={<Spec />} />
        <Route path="/welcome" element={<Welcome />} />
        <Route path="/account" element={<Account />} />
        <Route path="/agent" element={<Agent />} />
        <Route path="/contact" element={<Contact />} />

        {/* Legacy verification URLs embedded in receipts issued before the
            in-browser verifier was retired. Render a "moved" page that shows
            the hash and points at security@kitaab.live, instead of letting
            the SPA catchall dump auditors on the home page. */}
        <Route path="/verify" element={<VerifyMoved />} />
        <Route path="/verify/:hash" element={<VerifyMoved />} />

        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </Layout>
  );
}
