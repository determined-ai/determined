import { ExclamationCircleOutlined } from '@ant-design/icons';
import { Modal as AntdModal } from 'antd';
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
import Link from 'components/Link';
import Icon from 'shared/components/Icon';
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
  cancelText: string;
  danger?: boolean;
  footerLink?: LinkParams;
  headerLink?: LinkParams;
  icon?: string;
  key?: string;
  size: ModalSize;
  submit: ModalSubmitParams;
  titleText: string;
  children: ReactNode;
}

const ModalContext = createContext<ModalContext | null>(null);

export const Modal: React.FC<ModalProps> = ({
  cancelText,
  danger,
  footerLink,
  headerLink,
  icon,
  key,
  size = 'large',
  submit,
  titleText,
  children: modalBody,
}: ModalProps) => {
  const modalContext = useContext(ModalContext);

  if (modalContext === null) {
    throw new Error('Modal used outside of ModalContext');
  }
  const { isOpen, setIsOpen } = modalContext;

  const [isSubmitting, setIsSubmitting] = useState(false);

  const close = useCallback(() => setIsOpen(false), [setIsOpen]);

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
      footer={
        <div className={css.footer}>
          <div>
            {footerLink && (
              <Link path={footerLink.url} popout>
                {footerLink.text}
              </Link>
            )}
          </div>
          <div>
            <Button key="back" onClick={close}>
              {cancelText}
            </Button>
            <Button
              danger={danger}
              disabled={!!submit?.disabled}
              key="submit"
              loading={isSubmitting}
              type="primary"
              onClick={handleSubmit}>
              {submit.text}
            </Button>
          </div>
        </div>
      }
      key={key}
      maskClosable={true}
      open={isOpen}
      title={
        <div className={css.title}>
          {danger ? (
            <div className={`${css.dangerIcon} ${css.icon}`}>
              <ExclamationCircleOutlined />
            </div>
          ) : icon ? (
            <div className={css.icon}>
              <Icon name={icon} />
            </div>
          ) : null}
          <div className={css.titleText}>{titleText}</div>
          {headerLink && (
            <Link path={headerLink.url} popout>
              {headerLink.text}
            </Link>
          )}
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
