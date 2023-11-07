import { IconName } from 'hew/Icon';
import Nameplate from 'hew/Nameplate';
import React, { useCallback, useRef } from 'react';
import { CSSTransition } from 'react-transition-group';

import Link, { Props as LinkProps } from 'components/Link';

import css from './ActionSheet.module.scss';

export interface ActionItem extends LinkProps {
  icon?: IconName | React.ReactElement;
  label: string;
  render?: () => JSX.Element;
}

interface Props {
  actions: ActionItem[];
  hideCancel?: boolean;
  onCancel?: () => void;
  show?: boolean;
}

const ActionSheet: React.FC<Props> = ({ onCancel, ...props }: Props) => {
  const sheetRef = useRef<HTMLDivElement>(null);

  const handleOverlayClick = useCallback(
    (e: React.MouseEvent) => {
      // Prevent `onCancel` from getting called if the sheet (not the overlay) was clicked
      if (sheetRef.current?.contains(e.target as HTMLElement)) return;
      onCancel?.();
    },
    [onCancel],
  );

  const handleCancelClick = useCallback(() => {
    onCancel?.();
  }, [onCancel]);

  function renderActionItem(action: ActionItem) {
    if (action.render) {
      return action.render();
    } else {
      return (
        <Link className={css.item} key={action.label} path={action.path} {...action}>
          <Nameplate
            icon={action.icon ?? <span className={css.spacer} />}
            iconSize="large"
            name={action.label}
          />
        </Link>
      );
    }
  }

  return (
    <CSSTransition
      classNames={{
        enter: css.sheetEnter,
        enterActive: css.sheetEnterActive,
        enterDone: css.sheetEnterDone,
        exit: css.sheetExit,
        exitActive: css.sheetExitActive,
        exitDone: css.sheetExitDone,
      }}
      in={props.show}
      timeout={200}>
      <div className={css.base} onClick={handleOverlayClick}>
        <div className={css.sheet} ref={sheetRef}>
          <div className={css.actionList}>
            {props.actions.map((action, i) => (
              <React.Fragment key={action?.label ?? i}>{renderActionItem(action)}</React.Fragment>
            ))}
          </div>
          {!props.hideCancel && (
            <Link className={css.item} key="cancel" onClick={handleCancelClick}>
              <Nameplate icon="error" iconSize="large" name="Cancel" />
            </Link>
          )}
        </div>
      </div>
    </CSSTransition>
  );
};

export default ActionSheet;
