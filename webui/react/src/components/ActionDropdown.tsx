import { Dropdown, Menu, Modal, ModalFuncProps } from 'antd';
import { MenuInfo } from 'rc-menu/lib/interface';
import React, { JSXElementConstructor } from 'react';

import Icon from 'components/Icon';
import handleError, { ErrorLevel, ErrorType } from 'ErrorHandler';
import { Eventually } from 'types';
import { capitalize } from 'utils/string';

import css from './ActionDropdown.module.scss';

// TODO parameterize Action using Enums? https://github.com/microsoft/TypeScript/issues/30611
export type Triggers<T extends string> = Partial<{ [key in T]: () => Eventually<void> }>
export type Confirmations<T extends string> =
  Partial<{ [key in T]: Omit<ModalFuncProps, 'onOk'>}>

interface Props<T extends string> {
  actionOrder: T[];
  confirmations?: Confirmations<T>
  id: string;
  kind: string;
  onComplete?: (action?: T) => void;
  onTrigger: Triggers<T>;
}

const stopPropagation = (e: React.MouseEvent): void => e.stopPropagation();

const ActionDropdown = <T extends string>(
  { id, kind, onComplete, onTrigger, confirmations, actionOrder }: Props<T>,
): React.ReactElement<unknown, JSXElementConstructor<unknown>> | null => {

  const handleMenuClick = async (params: MenuInfo): Promise<void> => {
    params.domEvent.stopPropagation();
    try {
      const action = params.key as T;
      const handleTrigger = onTrigger[action];
      if (!handleTrigger) throw new Error(`No triggers for action ${action}`);
      const onOk = async () => {
        await handleTrigger();
        onComplete?.(action);
      };

      if (confirmations?.[action]) {
        Modal.confirm({
          content: `Are you sure you want to ${action.toLocaleLowerCase()} ${kind} "${id}"?`,
          title: `${capitalize(action)} ${kind}`,
          ...confirmations[action],
          onOk,
        });
      } else {
        await onOk();
      }

    } catch (e) {
      handleError({
        error: e,
        level: ErrorLevel.Error,
        message: e.message,
        publicMessage: `Unable to ${params.key} ${kind} ${id}.`,
        publicSubject: `${capitalize(params.key.toString())} failed.`,
        silent: false,
        type: ErrorType.Server,
      });
    }
  };

  const menuItems: React.ReactNode[] = actionOrder
    .filter(act => !!onTrigger[act])
    .map(action => <Menu.Item key={action}>{action}</Menu.Item>);

  if (menuItems.length === 0) {
    return (
      <div className={css.base} title="No actions available" onClick={stopPropagation}>
        <button disabled>
          <Icon name="overflow-vertical" />
        </button>
      </div>
    );
  }

  const menu = <Menu onClick={handleMenuClick}>{menuItems}</Menu>;

  return (
    <div className={css.base} title="Open actions menu" onClick={stopPropagation}>
      <Dropdown overlay={menu} placement="bottomRight" trigger={[ 'click' ]}>
        <button onClick={stopPropagation}>
          <Icon name="overflow-vertical" />
        </button>
      </Dropdown>
    </div>
  );
};

export default ActionDropdown;
