import { BaseComponent, NamedComponent } from 'e2e/models/BaseComponent';
import { Dropdown } from 'e2e/models/hew/Dropdown';

/**
 * Returns a representation of the OptionsMenu component.
 * This constructor represents the contents in src/components/OptionsMenu.tsx.
 * @param {object} obj
 * @param {CanBeParent} obj.parent - The parent used to locate this OptionsMenu
 * @param {string} obj.selector - Used instead of `defaultSelector`
 */
export class OptionsMenu extends NamedComponent {
  readonly defaultSelector = '[data-test-component="OptionsMenu"]';
  readonly dropdown = new Dropdown({
    parent: this._parent,
    selector: 'button' + this.defaultSelector,
  });
  readonly short = new BaseComponent({
    parent: this.dropdown._menu,
    selector: Dropdown.selectorTemplate('SHORT'),
  });
  readonly medium = new BaseComponent({
    parent: this.dropdown._menu,
    selector: Dropdown.selectorTemplate('MEDIUM'),
  });
  readonly tall = new BaseComponent({
    parent: this.dropdown._menu,
    selector: Dropdown.selectorTemplate('TALL'),
  });
  readonly extraTall = new BaseComponent({
    parent: this.dropdown._menu,
    selector: Dropdown.selectorTemplate('EXTRA_TALL'),
  });
}
