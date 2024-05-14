import { BaseComponent, NamedComponent } from 'e2e/models/BaseComponent';
import { ConjunctionContainer } from 'e2e/models/components/FilterForm/components/ConjunctionContainer';
import { FilterField } from 'e2e/models/components/FilterForm/components/FilterField';
import { Dropdown } from 'e2e/models/hew/Dropdown';

/**
 * Returns a representation of the FilterGroup component.
 * This constructor represents the contents in src/components/FilterGroup.tsx.
 * @param {object} obj
 * @param {CanBeParent} obj.parent - The parent used to locate this FilterGroup
 * @param {string} obj.selector - Used instead of `defaultSelector`
 */
export class FilterGroup extends NamedComponent {
  readonly defaultSelector = '[data-test-component="FilterGroup"]';

  #childrenSelector = '[data-test="children"]';
  #notNestedSelector = `:not(${this.#childrenSelector} *)`;
  private selectorTemplate = (selector: string) => `${selector}${this.#notNestedSelector}`;
  readonly conjunctionContainer = new ConjunctionContainer({ parent: this });
  readonly groupCard = new BaseComponent({
    parent: this,
    selector: this.selectorTemplate('[data-test="groupCard"]'),
  });
  readonly header = new BaseComponent({
    parent: this.groupCard,
    selector: this.selectorTemplate('[data-test="header"]'),
  });
  readonly explanation = new BaseComponent({
    parent: this.header,
    selector: this.selectorTemplate('[data-test="explanation"]'),
  });
  readonly addDropdown = new AddDropdown({
    parent: this.header,
    selector: this.selectorTemplate('[data-test="add"]'),
  });
  readonly remove = new BaseComponent({
    parent: this.header,
    selector: this.selectorTemplate('[data-test="remove"]'),
  });
  readonly move = new BaseComponent({
    parent: this.header,
    selector: this.selectorTemplate('[data-test="move"]'),
  });
  readonly children = new BaseComponent({
    parent: this.groupCard,
    selector: this.#childrenSelector,
  });
  readonly filterGroups = new FilterGroup({
    attachment: this.#notNestedSelector,
    parent: this.children,
  });
  readonly filterFields = new FilterField({
    attachment: this.#notNestedSelector,
    parent: this.children,
  });
}

class AddDropdown extends Dropdown {
  readonly addCondition = new BaseComponent({
    parent: this._menu,
    selector: Dropdown.selectorTemplate('field'),
  });
  readonly addConditionGroup = new BaseComponent({
    parent: this._menu,
    selector: Dropdown.selectorTemplate('group'),
  });
}
