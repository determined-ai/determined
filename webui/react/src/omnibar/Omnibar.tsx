import OmnibarNpm from 'omnibar';
import React, { useCallback, useEffect } from 'react';

import { StoreAction, useStore, useStoreDispatch } from 'contexts/Store';
import { exposeStore } from 'omnibar/exposedStore';
import * as Tree from 'omnibar/tree-extension/index';
import TreeNode from 'omnibar/tree-extension/TreeNode';
import { BaseNode } from 'omnibar/tree-extension/types';

import css from './Omnibar.module.scss';

const Omnibar: React.FC = () => {
  const storeDispatch = useStoreDispatch();
  const { ui } = useStore();

  const hideBar = useCallback(
    () => storeDispatch({ type: StoreAction.HideOmnibar }),
    [ storeDispatch ],
  );

  useEffect(() => {
    exposeStore({ dispatch: storeDispatch, state: { ui } });
  }, [ storeDispatch, ui ]);

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
          onAction={Tree.onAction as any} />
      </div>
    </div>
  );
};

export default Omnibar;
