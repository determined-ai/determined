import { Dropdown, Menu, Modal, ModalFuncProps } from 'antd';
import type { MenuProps } from 'antd';
import { MenuInfo } from 'rc-menu/lib/interface';
import React, { JSXElementConstructor, useCallback } from 'react';

import Icon from 'shared/components/Icon/Icon';
import { wrapPublicMessage } from 'shared/utils/error';
import { capitalize } from 'shared/utils/string';

import { Eventually } from '../../types';
import { DetError, ErrorLevel, ErrorType } from '../../utils/error';

import css from './ActionDropdown.module.scss';

// TODO parameterize Action using Enums? https://github.com/microsoft/TypeScript/issues/30611
export type Triggers<T extends string> = Partial<{ [key in T]: () => Eventually<void> }>
export type Confirmations<T extends string> =
  Partial<{ [key in T]: Omit<ModalFuncProps, 'onOk'> }>
type DisabledActions<T extends string> =
  Partial<{ [key in T]: boolean }>

interface Props<T extends string> {
  /**
   * define the order of actions to show up in the dropdown menu.
   */
  actionOrder: T[];
  /**
   * whether to prompt the user to confirm the action before executing it
   * with options to customize the generated modal.
   */
  confirmations?: Confirmations<T>
  /**
   * whether to disable the action or not.
   */
  disabled?: DisabledActions<T>;
  /**
   * How to identify the entity that the action is being performed on.
   * This is used to generate the modal content and for logging purposes.
   */
  id: string;
  /**
   * kind of the entity that the action is being performed on.
   */
  kind: string;
  /**
   * what to do after each action is completed.
   */
  onComplete?: (action?: T) => void;
  /**
   * how to handle errors.
   */
  onError: (error: DetError) => void;
  /**
   * what to do when an action is selected.
   */
  onTrigger: Triggers<T>;
}

const stopPropagation = (e: React.MouseEvent): void => e.stopPropagation();

const ActionDropdown = <T extends string>(
  { id, kind, onComplete, onTrigger, confirmations, disabled, actionOrder, onError }: Props<T>,
): React.ReactElement<unknown, JSXElementConstructor<unknown>> | null => {

  const menuClickErrorHandler = useCallback((
    e: unknown,
    actionKey: string,
    kind: string,
    id: string,
  ): void => {
    onError(new DetError(e, {
      level: ErrorLevel.Error,
      publicMessage: wrapPublicMessage(e, `Unable to ${actionKey} ${kind} ${id}.`),
      publicSubject: `${capitalize(actionKey.toString())} failed.`,
      silent: false,
      type: ErrorType.Server,
    }));
  }, [ onError ]);

  const handleMenuClick = async (params: MenuInfo): Promise<void> => {
    params.domEvent.stopPropagation();
    try {
      const action = params.key as T;
      const handleTrigger = onTrigger[action];
      if (!handleTrigger) throw new Error(`No triggers for action ${action}`);
      const onOk = async () => {
        try {
          await handleTrigger();
          onComplete?.(action);
        } catch (e) {
          menuClickErrorHandler(e, action, kind, id);
        }
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
      menuClickErrorHandler(e, params.key, kind, id);
    }
  };

  const menuItems: MenuProps['items'] = actionOrder
    .filter((act) => !!onTrigger[act])
    .map((action) => ({ disabled: disabled?.[action], key: action, label: action }));

  if (menuItems.length === 0) {
    return (
      <div className={css.base} title="No actions available" onClick={stopPropagation}>
        <button disabled>
          <Icon name="overflow-vertical" />
        </button>
      </div>
    );
  }

  return (
    <div className={css.base} title="Open actions menu" onClick={stopPropagation}>
      <Dropdown
        overlay={<Menu items={menuItems} onClick={handleMenuClick} />}
        placement="bottomRight"
        trigger={[ 'click' ]}>
        <button onClick={stopPropagation}>
          <Icon name="overflow-vertical" />
        </button>
      </Dropdown>
    </div>
  );
};

export default ActionDropdown;
