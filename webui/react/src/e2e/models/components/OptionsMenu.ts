import { BaseComponent } from 'e2e/models/BaseComponent';
import { BasePage } from 'e2e/models/BasePage';
import { DropdownMenu } from 'e2e/models/hew/Dropdown';

/**
 * Returns a representation of the OptionsMenu component.
 * This constructor represents the contents in src/components/OptionsMenu.tsx.
 * @param {object} obj
 * @param {CanBeParent} obj.parent - The parent used to locate this OptionsMenu
 * @param {string} obj.selector - Used instead of `defaultSelector`
 */
export class OptionsMenu extends DropdownMenu {
  constructor({ parent, root }: { parent: BaseComponent; root: BasePage }) {
    super({
      childNode: new BaseComponent({ parent, selector: '[data-test-component="OptionsMenu"]' }),
      root,
    });
  }
  readonly defaultSelector = '[data-test-component="OptionsMenu"]';
  readonly short = this.menuItem('SHORT');
  readonly medium = this.menuItem('MEDIUM');
  readonly tall = this.menuItem('TALL');
  readonly extraTall = this.menuItem('EXTRA_TALL');
}
