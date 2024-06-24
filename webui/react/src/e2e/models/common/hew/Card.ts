import {
  BaseComponent,
  CanBeParent,
  NamedComponent,
  NamedComponentArgs,
} from 'e2e/models/common/base/BaseComponent';
import { WorkspaceActionDropdown } from 'e2e/models/components/WorkspaceActionDropdown';

import { DropdownMenu } from './Dropdown';

/**
 * Represents the Card component in hew/src/kit/Card.tsx
 */
export class Card extends NamedComponent {
  override defaultSelector: string = ''; // must be provided
  // provide an actionMenu with a Dropdown to use
  static actionMenuSelector = '[aria-label="Action menu"]';

  // default to a workspace dropdown to avoid non-null but this should be overriden if a dropdown exists
  readonly actionMenu: DropdownMenu = new WorkspaceActionDropdown({
    clickThisComponentToOpen: new BaseComponent({
      parent: this,
      selector: Card.actionMenuSelector,
    }),
    root: this.root,
  });

  static withName<T extends Card>(
    props: { name: string; parent: CanBeParent },
    cardType: new (args: NamedComponentArgs) => T,
  ): T {
    return new cardType({
      parent: props.parent,
      selector: `[data-testid="card-${props.name}"]`,
    });
  }
}
