import OmnibarNpm from 'omnibar';
import React from 'react';
import { GlobalHotKeys } from 'react-hotkeys';
import OmnibarCtx from 'omnibar/Context';

import * as Tree from 'omnibar/Tree';

import css from './Component.module.scss';

interface ItemProps<T> {
  item: T;
  isSelected: boolean;
  isHighlighted: boolean;
}

const ItemRenderer = (p: ItemProps<any>) => {
  return (
    <div>{p.item.name}</div>
  );
};

const globalKeymap = {
  HIDE_OMNIBAR: [ 'esc' ], // TODO scope it to the component
};


const Omnibar: React.FC = () => {
  const setOmnibar = OmnibarCtx.useActionContext();
  const globalKeyHandler = {
    HIDE_OMNIBAR: (): void => {
      setOmnibar({ type: OmnibarCtx.ActionType.Hide })
      alert('esc pressed')
    },
    SHOW_OMNIBAR: (): void => setOmnibar({ type: OmnibarCtx.ActionType.Show }),
  };
  return (
    <div className={css.base}>
      <div className={css.bar} id="omnibar">
        {/* <OmnibarNpm
        autoFocus={true}
        extensions={[ funcExt ]}
        placeholder="Type a function name"
        onAction={funcOnAction}
        /> */}
        <OmnibarNpm
          autoFocus={true}
          extensions={[ Tree.extension ]}
          maxResults={7}
          placeholder="Type away.."
          onAction={Tree.onAction}
        /*render={ItemRenderer}*/ />
      </div>
    <GlobalHotKeys handlers={globalKeyHandler} keyMap={globalKeymap} />
    </div>
  );
};

export const keymap = { HIDE_OMNIBAR: 'esc' };

export default Omnibar;
