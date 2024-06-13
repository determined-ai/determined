import { BaseComponent } from 'e2e/models/BaseComponent';
import { BasePage } from 'e2e/models/BasePage';
import { DropdownMenu } from 'e2e/models/hew/Dropdown';

/**
 * Represents the OptionsMenu component in src/components/FilterForm/OptionsMenu.tsx
 */
export class OptionsMenu extends DropdownMenu {
  /**
   * Constructs a OptionsMenu
   * @param {object} obj
   * @param {CanBeParent} obj.parent - parent component
   * @param {BasePage} obj.root - root page
   */
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
