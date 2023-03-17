import { ExclamationCircleOutlined } from '@ant-design/icons';
import { Modal as AntdModal } from 'antd';
import React, {
  createContext,
  Dispatch,
  ReactNode,
  SetStateAction,
  useCallback,
  useContext,
  useEffect,
  useState,
} from 'react';

import Button from 'components/kit/Button';
import Link from 'components/Link';
import Icon from 'shared/components/Icon';

import css from './useModality.module.scss';

interface LinkParams {
  text: string;
  url: string;
}

type ModalSize = 'small' | 'medium' | 'large';

export interface ModalSubmitParams {
  disabled?: boolean;
  text: string;
  handler: () => Promise<void>;
  onComplete?: () => Promise<void>;
}

export interface UseModalParams {
  cancelText: string;
  danger?: boolean;
  footerLink?: LinkParams;
  headerLink?: LinkParams;
  icon?: string;
  key?: string;
  size: ModalSize;
  submit: ModalSubmitParams;
  titleText: string;
}

interface ModalContext {
  modalIsOpen: boolean;
  params?: UseModalParams;
  setParams: Dispatch<SetStateAction<UseModalParams | null>>;
}

const ModalContext = createContext<ModalContext | null>(null);

export function useModalParams(params: UseModalParams): void {
  const modalContext = useContext(ModalContext);
  if (modalContext === null) throw new Error('tried to use modal context outside of modal');
  const { setParams } = modalContext;
  useEffect(() => setParams(params), [setParams, params]);
}

interface ModalContainerProps {
  modalIsOpen: boolean;
  setModalIsOpen: Dispatch<SetStateAction<boolean>>;
  children: ReactNode;
}

function ModalContainer({ modalIsOpen, children: modalBody, setModalIsOpen }: ModalContainerProps) {
  const [params, setParams] = useState<UseModalParams | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const close = useCallback(() => setModalIsOpen(false), [setModalIsOpen]);

  const handleOk = useCallback(async () => {
    setIsSubmitting(true);
    await params?.submit?.handler();
    setIsSubmitting(false);
    setModalIsOpen(false);
    await params?.submit?.onComplete?.();
  }, [params?.submit, setModalIsOpen]);

  if (params === null) {
    return (
      <ModalContext.Provider value={{ modalIsOpen, setParams }}>
        <div style={{ display: 'none' }}>{modalBody}</div>
      </ModalContext.Provider>
    );
  }

  const { titleText, cancelText, submit, danger, icon, footerLink, headerLink, size } = params;
  return (
    <ModalContext.Provider value={{ modalIsOpen, params, setParams }}>
      <AntdModal
        cancelText={params.cancelText}
        footer={
          <div style={{ display: 'flex', justifyContent: 'space-between' }}>
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
                onClick={handleOk}>
                {submit.text}
              </Button>
            </div>
          </div>
        }
        maskClosable={true}
        open={modalIsOpen}
        style={{
          minWidth: size === 'small' ? 358 : size === 'medium' ? 692 : 1025,
        }}
        title={
          <div style={{ display: 'flex' }}>
            {danger ? (
              <div className={css.dangerIcon} style={{ paddingRight: 16 }}>
                <ExclamationCircleOutlined />
              </div>
            ) : icon ? (
              <div style={{ paddingRight: 16 }}>
                <Icon name={icon} />
              </div>
            ) : null}
            <div style={{ paddingRight: 4 }}>{titleText}</div>
            {headerLink && (
              <Link path={headerLink.url} popout>
                {headerLink.text}
              </Link>
            )}
          </div>
        }
        onCancel={close}
        onOk={handleOk}>
        {modalBody}
      </AntdModal>
    </ModalContext.Provider>
  );
}
export function useModalComponent<ModalProps extends JSX.IntrinsicAttributes>(
  ModalBodyComponent: React.FC<ModalProps>,
): { Component: React.FC<ModalProps>; open: () => void } {
  const [modalIsOpen, setModalIsOpen] = useState(false);
  const handleOpen = useCallback(() => setModalIsOpen(true), []);

  const Component = useCallback(
    (props: ModalProps) => {
      return (
        <ModalContainer modalIsOpen={modalIsOpen} setModalIsOpen={setModalIsOpen}>
          <ModalBodyComponent {...props} />
        </ModalContainer>
      );
    },
    [ModalBodyComponent, modalIsOpen],
  );
  return { Component, open: handleOpen };
}
