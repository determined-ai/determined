import { Modal as AntdModal, ModalProps as AntdModalProps } from 'antd';
import React, {
  createContext,
  Dispatch,
  ReactNode,
  SetStateAction,
  useCallback,
  useContext,
  useState,
} from 'react';

import Button from 'components/kit/Button';
import Icon, { IconName } from 'components/kit/Icon';
import Link from 'components/Link';
import Spinner from 'shared/components/Spinner';
import { ErrorLevel, ErrorType } from 'shared/utils/error';
import handleError from 'utils/error';

import css from './Modal.module.scss';

interface LinkParams {
  text: string;
  url: string;
}

export type ModalSize = 'small' | 'medium' | 'large';
const modalWidths: { [key in ModalSize]: number } = {
  large: 1025,
  medium: 692,
  small: 358,
};

export type Opener = Dispatch<SetStateAction<boolean>>;

export type ModalContext = {
  isOpen: boolean;
  setIsOpen: Opener;
};

export interface ModalSubmitParams {
  disabled?: boolean;
  text: string;
  handler: () => Promise<void>;
  onComplete?: () => Promise<void>;
}

interface ModalProps {
  cancel?: boolean;
  cancelText?: string;
  danger?: boolean;
  footerLink?: LinkParams;
  headerLink?: LinkParams;
  icon?: IconName;
  key?: string;
  onClose?: () => void;
  size?: ModalSize;
  submit?: ModalSubmitParams;
  title: string;
  okButtonProps?: AntdModalProps['okButtonProps'];
  children: ReactNode;
}

export const DEFAULT_CANCEL_LABEL = 'Cancel';

const ModalContext = createContext<ModalContext | null>(null);

export const Modal: React.FC<ModalProps> = ({
  cancel,
  cancelText,
  danger,
  footerLink,
  headerLink,
  icon,
  key,
  onClose,
  size = 'large',
  submit,
  title,
  okButtonProps,
  children: modalBody,
}: ModalProps) => {
  const modalContext = useContext(ModalContext);

  if (modalContext === null) {
    throw new Error('Modal used outside of ModalContext');
  }
  const { isOpen, setIsOpen } = modalContext;

  const [isSubmitting, setIsSubmitting] = useState(false);

  const close = useCallback(() => {
    setIsOpen(false);
    onClose?.();
  }, [setIsOpen, onClose]);

  const handleSubmit = useCallback(async () => {
    setIsSubmitting(true);
    try {
      await submit?.handler();
      setIsSubmitting(false);
      setIsOpen(false);
      await submit?.onComplete?.();
    } catch (err) {
      handleError(err, {
        level: ErrorLevel.Error,
        publicMessage: err instanceof Error ? err.message : '',
        publicSubject: 'Could not submit form',
        silent: false,
        type: ErrorType.Server,
      });
      setIsSubmitting(false);
    }
  }, [submit, setIsOpen]);

  return (
    <AntdModal
      cancelText={cancelText}
      className={css.modalContent}
      closeIcon={<Icon name="close" size="small" />}
      footer={
        <div className={css.footer}>
          <div className={css.footerLink}>
            {footerLink && (
              <Link path={footerLink.url} popout>
                {footerLink.text}
              </Link>
            )}
          </div>
          <div className={css.buttons}>
            {(cancel || cancelText) && (
              <Button key="back" onClick={close}>
                {cancelText || DEFAULT_CANCEL_LABEL}
              </Button>
            )}
            <Button
              danger={danger}
              disabled={!!submit?.disabled}
              key="submit"
              loading={isSubmitting}
              tooltip={submit?.disabled ? 'Address validation errors before proceeding' : undefined}
              type="primary"
              {...okButtonProps}
              onClick={handleSubmit}>
              {submit?.text ?? 'OK'}
            </Button>
          </div>
        </div>
      }
      key={key}
      maskClosable={true}
      open={isOpen}
      title={
        <div className={css.header}>
          {danger ? (
            <div className={css.dangerIcon}>
              <Icon name="warning-large" size="large" />
            </div>
          ) : (
            icon && <Icon name={icon} size="large" />
          )}
          <div className={css.headerTitle}>{title}</div>
          <div className={css.headerLink}>
            {headerLink && (
              <Link path={headerLink.url} popout>
                {headerLink.text}
              </Link>
            )}
          </div>
        </div>
      }
      width={modalWidths[size]}
      onCancel={close}
      onOk={handleSubmit}>
      <Spinner spinning={isSubmitting}>
        <div className={css.modalBody}>{modalBody}</div>
      </Spinner>
    </AntdModal>
  );
};

export const useModal = <ModalProps extends object>(
  Comp: React.FC<ModalProps>,
): { Component: React.FC<ModalProps>; open: () => void } => {
  const [isOpen, setIsOpen] = useState(false);
  const handleOpen = React.useCallback(() => setIsOpen(true), []);

  const Component = React.useCallback(
    (props: ModalProps) => {
      return (
        <ModalContext.Provider value={{ isOpen, setIsOpen }}>
          <Comp {...props} />
        </ModalContext.Provider>
      );
    },
    [Comp, isOpen],
  );
  return { Component, open: handleOpen };
};
