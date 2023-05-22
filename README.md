# WebTerm

An integrated terminal for your browser.

![screenshot](./assets/screenshot.png)

## Installation

```bash
# clone the repository
git clone https://github.com/pomdtr/webterm && cd webterm

# install the cli
go install
webterm init

# build the extension
cd extension
npm run build
```

Then go to the `chrome://extensions` page, activate the Developer mode and click on the `Load unpacked`. 

You will need to select the `extension/dist` folder using the file picker.
