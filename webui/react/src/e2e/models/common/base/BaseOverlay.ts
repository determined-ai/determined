import { BaseComponent, ComponentBasics } from 'e2e/models/common/base/BaseComponent';
import { BasePage } from 'e2e/models/common/base/BasePage';

// typically, we would have a clickThisComponentToOpen or an openMethod, but not both
// in exceptional cases, we might have both, like when working with canvas elements
// in other cases, we might have neither, like a modal which has multiple ways to open
export interface OverlayArgs {
  root: BasePage;
  clickThisComponentToOpen?: ComponentBasics;
  openMethod?: () => Promise<void>;
}

/**
 * Represents a Base Overlay component
 */
export abstract class BaseOverlay extends BaseComponent {
  readonly openMethod: (args: object) => Promise<void>;
  readonly clickThisComponentToOpen?: ComponentBasics;

  /**
   * Constructs a new overlay component.
   * The overlay can be opened by calling the open method. By default, the open
   * method clicks on the child node. Sometimes you might even need to provide
   * both optional arguments, like when a child node is present but impossible to
   * click on due to being blocked by another element behavior.
   * @param {object} obj
   * @param {string} obj.selector - the selector for the overlay
   * @param {BasePage} obj.root - root of the page
   * @param {ComponentBasics} [obj.clickThisComponentToOpen] - optional if `openMethod` is present. It's the element we click on to open the dropdown.
   * @param {Function} [obj.openMethod] - optional if `clickThisComponentToOpen` is present. It's the method to open the dropdown.
   */
  constructor({
    selector,
    root,
    clickThisComponentToOpen,
    openMethod,
  }: OverlayArgs & { selector: string }) {
    super({
      parent: root,
      selector,
    });
    if (clickThisComponentToOpen !== undefined) {
      this.clickThisComponentToOpen = clickThisComponentToOpen;
    }
    this.openMethod =
      openMethod ||
      (async (args = {}) => {
        if (this.clickThisComponentToOpen === undefined) {
          // We should never be able to throw this error. In the constructor, we
          // either provide a clickThisComponentToOpen or replace this method.
          throw new Error('This popover does not have a child node to click on.');
        }
        await this.clickThisComponentToOpen.pwLocator.click(args);
        await this.pwLocator.waitFor();
      });
  }

  /**
   * Opens the overlay
   * @returns {Promise<this>} - the popover for further actions
   */
  async open(args = {}): Promise<this> {
    await this.openMethod(args);
    return this;
  }

  /**
   * Closes the overlay
   */
  abstract close(): Promise<void>;
}
