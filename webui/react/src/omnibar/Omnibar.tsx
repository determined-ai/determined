import OmnibarNpm from 'omnibar';
import React, { useCallback, useEffect } from 'react';
import { GlobalHotKeys } from 'react-hotkeys';

import { StoreAction, useStore, useStoreDispatch } from 'contexts/Store';
import { exposeStore } from 'omnibar/exposedStore';
import * as Tree from 'omnibar/tree-extension/index';
import TreeNode from 'omnibar/tree-extension/TreeNode';
import { BaseNode } from 'omnibar/tree-extension/types';

import css from './Omnibar.module.scss';

const globalKeymap = { HIDE_OMNIBAR: [ 'esc' ] };

const Omnibar: React.FC = () => {
  const storeDispatch = useStoreDispatch();
  const { omnibar } = useStore();

  const hideBar = useCallback(
    () => storeDispatch({ type: StoreAction.HideOmnibar }),
    [ storeDispatch ],
  );
  const globalKeyHandler = {
    HIDE_OMNIBAR: hideBar,
    SHOW_OMNIBAR: (): void => storeDispatch({ type: StoreAction.ShowOmnibar }),
  };

  useEffect(() => {
    exposeStore({ dispatch: storeDispatch, state: { omnibar } });
  }, [ storeDispatch, omnibar ]);

  return (
    <div className={css.base} onClick={hideBar}>
      <div className={css.bar} id="omnibar">
        <OmnibarNpm<BaseNode>
          autoFocus={true}
          extensions={[ Tree.extension ]}
          maxResults={7}
          placeholder='Type a command or "help" for more info.'
          render={TreeNode}
          onAction={Tree.onAction as any} />
      </div>
      <GlobalHotKeys handlers={globalKeyHandler} keyMap={globalKeymap} />
    </div>
  );
};

export const keymap = { HIDE_OMNIBAR: 'esc' };

export default Omnibar;
