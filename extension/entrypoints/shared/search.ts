import { ITerminalAddon, Terminal } from '@xterm/xterm';
import { ISearchOptions, SearchAddon } from '@xterm/addon-search';
import './search.css';

export interface SearchBarOption extends ISearchOptions {
    searchAddon: SearchAddon;
}

const ADDON_MARKER_NAME = 'xterm-search-bar__addon';

export class SearchBarAddon implements ITerminalAddon {
    private readonly options: Partial<SearchBarOption>;
    // @ts-ignore
    private terminal: Terminal;
    // @ts-ignore
    private readonly searchAddon: SearchAddon;
    // @ts-ignore
    private searchBarElement: HTMLDivElement;
    // @ts-ignore
    private searchKey: string;

    constructor(options: Partial<SearchBarOption>) {
        this.options = options || {};
        if (this.options && this.options.searchAddon) {
            this.searchAddon = this.options.searchAddon;
        }
    }

    activate(terminal: Terminal): void {
        this.terminal = terminal;
        if (!this.searchAddon) {
            console.error('Cannot use search bar addon until search addon has been loaded!');
        }
    }

    dispose() {
        this.hidden();
    }

    /**
     *  Show the bar in the term
     * @returns empty
     * @memberof SearchBarAddon  necessary search addon instance
     */
    show() {
        if (!this.terminal || !this.terminal.element) {
            console.error('Cannot show search bar addon until terminal has been initialized!');
            return;
        }
        if (this.searchBarElement) {
            this.searchBarElement.style.visibility = 'visible';
            (this.searchBarElement.querySelector('input') as HTMLInputElement).select();
            return;
        }

        this.terminal.element.style.position = 'relative';
        const element = document.createElement('div');
        element.innerHTML = `
       <input type="text" class="search-bar__input" name="search-bar__input"/>
       <button class="search-bar__btn prev"></button>
       <button class="search-bar__btn next"></button>
       <button class="search-bar__btn close"></button>
    `;
        element.className = ADDON_MARKER_NAME;
        const parentElement = <HTMLElement>this.terminal.element.parentElement;
        this.searchBarElement = element;
        if (!['relative', 'absoulte', 'fixed'].includes(parentElement.style.position)) {
            parentElement.style.position = 'relative';
        }
        parentElement.appendChild(this.searchBarElement);
        this.on('.search-bar__btn.close', 'click', () => {
            this.hidden();
        });
        this.on('.search-bar__btn.next', 'click', () => {
            this.searchAddon.findNext(this.searchKey, {
                incremental: false
            });
        });
        this.on('.search-bar__btn.prev', 'click', () => {
            this.searchAddon.findPrevious(this.searchKey, {
                incremental: false
            });
        });
        this.on('.search-bar__input', 'keyup', (e: any) => {
            this.searchKey = (e.target as HTMLInputElement).value;
            this.searchAddon.findNext(this.searchKey, {
                incremental: e.key !== `Enter`
            });
        });
        (this.searchBarElement.querySelector('input') as HTMLInputElement).select();
    }

    /**
     * You can manually call close, also can click the close button on the bar
     * @memberof SearchBarAddon
     */
    hidden() {
        if (this.searchBarElement && (this.terminal.element as HTMLElement).parentElement) {
            this.searchBarElement.style.visibility = 'hidden';
        }
    }

    private on(selector: string, event: string, cb: (e: any) => void) {
        const parentElement = <HTMLElement>(this.terminal.element as HTMLElement).parentElement;
        parentElement.addEventListener(event, (e) => {
            let target = e.target;

            while (target !== document.querySelector(selector)) {
                if (target === parentElement) {
                    target = null;
                    break;
                }

                target = (target as HTMLElement).parentElement;
            }

            if (target === document.querySelector(selector)) {
                cb.call(this, e);
                e.stopPropagation();
            }
        });
    }

    /**
     * You can customize your own style, and then add CSS string template after search bar init
     * @param {string} newStyle
     * @memberof SearchBarAddon
     */
    addNewStyle(newStyle: string) {
        let styleElement = document.getElementById(ADDON_MARKER_NAME) as HTMLStyleElement;

        if (!styleElement) {
            styleElement = document.createElement('style');
            styleElement.type = 'text/css';
            styleElement.id = ADDON_MARKER_NAME;
            document.getElementsByTagName('head')[0].appendChild(styleElement);
        }

        styleElement.appendChild(document.createTextNode(newStyle));
    }
}
