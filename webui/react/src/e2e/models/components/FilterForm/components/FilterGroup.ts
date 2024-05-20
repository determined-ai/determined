import { BaseComponent, NamedComponent, NamedComponentArgs } from 'e2e/models/BaseComponent';
import { ConjunctionContainer } from 'e2e/models/components/FilterForm/components/ConjunctionContainer';
import { FilterField } from 'e2e/models/components/FilterForm/components/FilterField';
import { DropdownMenu } from 'e2e/models/hew/Dropdown';

/**
 * Returns a representation of the FilterGroup component.
 * This constructor represents the contents in src/components/FilterGroup.tsx.
 * @param {object} obj
 * @param {CanBeParent} obj.parent - The parent used to locate this FilterGroup
 * @param {string} obj.selector - Used instead of `defaultSelector`
 * @param {number} [level] - Level of the FilterGroup. Max depth is 2.
 */
export class FilterGroup extends NamedComponent {
  readonly defaultSelector = '[data-test-component="FilterGroup"]';
  constructor(args: NamedComponentArgs & { level?: number }) {
    super(args);
    const level = args.level || 0;
    if (level < 2) {
      // UI supports up to 2 levels of nesting
      this.filterGroups = new FilterGroup({
        attachment: this.#notNestedSelector,
        level: level + 1,
        parent: this.children,
      });
    }
  }

  readonly #childrenSelector = '[data-test="children"]';
  readonly #notNestedSelector = `:not(${this.#childrenSelector} *)`;

  /**
   * Ensures that the selector is not nested within the children selector.
   * @param selector the selector to use in the template
   * @returns the same selector with the not nested selector appended
   */
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
    childNode: new BaseComponent({
      parent: this.header,
      selector: this.selectorTemplate('[data-test="add"]'),
    }),
    root: this.root,
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
  readonly filterGroups: FilterGroup | undefined;
  readonly filterFields = new FilterField({
    attachment: this.#notNestedSelector,
    parent: this.children,
  });
}

class AddDropdown extends DropdownMenu {
  readonly addCondition = this.menuItem('field');
  readonly addConditionGroup = this.menuItem('group');
}
