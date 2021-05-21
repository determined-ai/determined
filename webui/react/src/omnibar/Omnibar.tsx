import OmnibarNpm from 'omnibar';
import React, { useCallback } from 'react';

import { StoreAction, useStoreDispatch } from 'contexts/Store';
import * as Tree from 'omnibar/tree-extension/index';
import TreeNode from 'omnibar/tree-extension/TreeNode';
import { BaseNode } from 'omnibar/tree-extension/types';
import { isTreeNode } from 'omnibar/tree-extension/utils';

import css from './Omnibar.module.scss';

const Omnibar: React.FC = () => {
  const storeDispatch = useStoreDispatch();

  const hideBar = useCallback(
    () => storeDispatch({ type: StoreAction.HideOmnibar }),
    [ storeDispatch ],
  );

  const onAction = useCallback((item, query) => {
    /*
    Ideally we wouldn't need to access the element like this.
    A potential option is use the value prop in combinatio with encoding the tree path into
    the options returned by the tree extension.
    */
    const input: HTMLInputElement|null = document.querySelector('#omnibar input[type="text"]');
    if (!input) return;
    if (isTreeNode(item)) {
      return Tree.onAction(input, item, query);
    }
  }, []);

  return (
    <div className={css.base}>
      <div className={css.backdrop} onClick={hideBar} />
      <div className={css.bar} id="omnibar">
        <OmnibarNpm<BaseNode>
          autoFocus={true}
          extensions={[ Tree.extension ]}
          maxResults={7}
          placeholder='Type a command or "help" for more info.'
          render={TreeNode}
          onAction={onAction} />
      </div>
    </div>
  );
};

export default Omnibar;
