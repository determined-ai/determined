import React, { useCallback, useRef } from 'react';
import { CSSTransition } from 'react-transition-group';

import css from './ActionSheet.module.scss';
import Icon from './Icon';
import Link, { Props as LinkProps } from './Link';

export interface ActionItem extends LinkProps {
  icon?: string;
  label: string;
  popout?: boolean;
}

interface Props {
  actions: ActionItem[];
  hideCancel?: boolean;
  onCancel?: () => void;
  show?: boolean;
}

const ActionSheet: React.FC<Props> = ({ onCancel, ...props }: Props) => {
  const sheetRef = useRef<HTMLDivElement>(null);

  const handleOverlayClick = useCallback((e: React.MouseEvent) => {
    // Prevent `onCancel` from getting called if the sheet (not the overlay) was clicked
    if (sheetRef.current && sheetRef.current.contains(e.target as HTMLElement)) return;
    if (onCancel) onCancel();
  }, [ onCancel ]);

  const handleCancelClick = useCallback(() => {
    if (onCancel) onCancel();
  }, [ onCancel ]);

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
          {props.actions.map(action => (
            <Link className={css.item} key={action.label} path={action.path} {...action}>
              {action.icon && <Icon name={action.icon} size="large" />}
              <div className={css.label}>{action.label}</div>
            </Link>
          ))}
          {!props.hideCancel && <Link className={css.item} onClick={handleCancelClick}>
            <Icon name="error" size="large" />
            <div className={css.label}>Cancel</div>
          </Link>}
        </div>
      </div>
    </CSSTransition>
  );
};

export default ActionSheet;
