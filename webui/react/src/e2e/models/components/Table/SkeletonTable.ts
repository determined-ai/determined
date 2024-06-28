import { BaseComponent } from 'e2e/models/common/base/BaseComponent';
import { SkeletonSection } from 'e2e/models/components/SkeletonSection';

/**
 * Represents the SkeletonTable component in src/components/Table/SkeletonTable.tsx
 */
export class SkeletonTable extends SkeletonSection {
  readonly table = new BaseComponent({
    parent: this,
    selector: '[data-testid="skeletonTable"]',
  });
}
