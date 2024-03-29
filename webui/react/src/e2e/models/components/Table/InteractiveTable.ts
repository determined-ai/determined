import { NamedComponent, NamedComponentArgs } from 'e2e/models/BaseComponent';
import { SkeletonTable } from 'e2e/models/components/Table/SkeletonTable';

/**
 * Returns a representation of the InteractiveTable component.
 * This constructor represents the contents in src/components/Table/InteractiveTable.tsx.
 * @param {object} obj
 * @param {implementsGetLocator} obj.parent - The parent used to locate this InteractiveTable
 * @param {string} [obj.selector] - Used instead of `defaultSelector`
 */
export class InteractiveTable extends NamedComponent {
  static defaultSelector = `div[data-test-component="interactiveTable"]`;
  constructor({ selector, parent }: NamedComponentArgs) {
    super({ parent: parent, selector: selector || InteractiveTable.defaultSelector });
  }

  readonly skeleton: SkeletonTable = new SkeletonTable({ parent: this });
  // readonly table: Table = new Table({ parent: this });
}
