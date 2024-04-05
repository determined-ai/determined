import { BaseComponent } from 'e2e/models/BaseComponent';
import { SkeletonSection } from 'e2e/models/components/SkeletonSection';

/**
 * Returns a representation of the SkeletonTable component.
 * This constructor represents the contents in src/components/Table/SkeletonTable.tsx.
 * @param {object} obj
 * @param {implementsGetLocator} obj.parent - The parent used to locate this SkeletonTable
 * @param {string} [obj.selector] - Used instead of `defaultSelector`
 */
export class SkeletonTable extends SkeletonSection {
  readonly table: BaseComponent = new BaseComponent({
    parent: this,
    selector: '[data-testid="skeletonTable"]',
  });
}
