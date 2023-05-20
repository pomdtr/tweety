import logo from "~/assets/logo.svg";
import "./style.css";

const imageUrl = new URL(logo, import.meta.url).href;

let count = 0;

document.querySelector("#app")!.innerHTML = `
  <img src="${imageUrl}" height="45" alt="" />
  <h1>Options</h1>
  <button type="button">Clicks: ${count}</button>
`;

const buttonElement = document.querySelector("button")!;
buttonElement.addEventListener("click", () => {
  count += 1;

  buttonElement.textContent = `Clicks: ${count}`;
});
