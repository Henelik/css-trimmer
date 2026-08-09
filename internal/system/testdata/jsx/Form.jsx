import React from "react";

export default function Form() {
  return (
    <form className="container">
      <label htmlFor="username">Username</label>
      <input id="username" className="input" type="text" placeholder="Enter username" />
      <button className="btn" type="submit">Submit</button>
    </form>
  );
}
