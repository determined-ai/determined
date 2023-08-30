import React, { JSXElementConstructor, useCallback } from 'react';

import css from 'components/ActionDropdown/ActionDropdown.module.scss';
import Button from 'components/kit/Button';
import Dropdown, { MenuItem } from 'components/kit/Dropdown';
import Icon from 'components/kit/Icon';
import useConfirm, { ConfirmModalProps } from 'components/kit/useConfirm';
import { Eventually } from 'types';
import handleError, { DetError, ErrorLevel, ErrorType, wrapPublicMessage } from 'utils/error';
import { capitalize } from 'utils/string';

// TODO parameterize Action using Enums? https://github.com/microsoft/TypeScript/issues/30611
export type Triggers<T extends string> = Partial<{ [key in T]: () => Eventually<void> }>;
export type Confirmations<T extends string> = Partial<{
  [key in T]: Omit<ConfirmModalProps, 'onConfirm'>;
}>;
type DisabledActions<T extends string> = Partial<{ [key in T]: boolean }>;
type DangerousActions<T extends string> = DisabledActions<T>;

interface Props<T extends string> {
  /**
   * define the order of actions to show up in the dropdown menu.
   */
  actionOrder: T[];
  children?: React.ReactNode;
  /**
   * whether to prompt the user to confirm the action before executing it
   * with options to customize the generated modal.
   */
  confirmations?: Confirmations<T>;
  /**
   * whether the action is marked as dangerous or not.
   */
  danger?: DangerousActions<T>;
  /**
   * whether to disable the action or not.
   */
  disabled?: DisabledActions<T>;
  /**
   * How to identify the entity that the action is being performed on.
   * This is used to generate the modal content and for logging purposes.
   */
  id: string;
  isContextMenu?: boolean;
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

const ActionDropdown = <T extends string>({
  id,
  isContextMenu,
  kind,
  onComplete,
  onTrigger,
  confirmations,
  danger,
  disabled,
  actionOrder,
  onError,
  children,
}: Props<T>): React.ReactElement<unknown, JSXElementConstructor<unknown>> | null => {
  const confirm = useConfirm();

  const menuClickErrorHandler = useCallback(
    (e: unknown, actionKey: string, kind: string, id: string): void => {
      onError(
        new DetError(e, {
          level: ErrorLevel.Error,
          publicMessage: wrapPublicMessage(e, `Unable to ${actionKey} ${kind} ${id}.`),
          publicSubject: `${capitalize(actionKey.toString())} failed.`,
          silent: false,
          type: ErrorType.Server,
        }),
      );
    },
    [onError],
  );

  const menuItems: MenuItem[] = actionOrder
    .filter((act) => !!onTrigger[act])
    .map((action) => ({
      danger: danger?.[action],
      disabled: disabled?.[action],
      key: action,
      label: action,
    }));

  if (menuItems.length === 0) {
    return (
      <div className={css.base} title="No actions available" onClick={stopPropagation}>
        <Button
          disabled
          icon={<Icon name="overflow-vertical" size="small" title="Action menu" />}
          type="text"
        />
      </div>
    );
  }

  const handleDropdown = async (key: string): Promise<void> => {
    try {
      const action = key as T;
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
        confirm({
          content: `Are you sure you want to ${action.toLocaleLowerCase()} ${kind} "${id}"?`,
          onConfirm: onOk,
          onError: handleError,
          title: `${capitalize(action)} ${kind}`,
          ...confirmations[action],
        });
      } else {
        await onOk();
      }
    } catch (e) {
      menuClickErrorHandler(e, key, kind, id);
    }
  };

  return children ? (
    <>
      <Dropdown
        isContextMenu={isContextMenu ?? true}
        menu={menuItems}
        placement="bottomRight"
        onClick={handleDropdown}>
        {children}
      </Dropdown>
    </>
  ) : (
    <div className={css.base} title="Open actions menu">
      <Dropdown menu={menuItems} placement="bottomRight" onClick={handleDropdown}>
        <Button icon={<Icon name="overflow-vertical" title="Action menu" />} type="text" />
      </Dropdown>
    </div>
  );
};

export default ActionDropdown;
