import React from "react";

export default function App() {
  return (
    <div className="container">
      <header className="header">
        <nav className="nav">
          <a className="nav-item" href="/">Home</a>
          <a className="nav-item" href="/about">About</a>
        </nav>
      </header>
      <main className="main-content">
        <div className="card">
          <h2 className="card-title">Card Title</h2>
          <button className="btn">Click me</button>
        </div>
      </main>
      <footer className="footer">
        <p>&copy; 2026</p>
      </footer>
    </div>
  );
}
