import { InputRef as AntdInputRef, InputRef, RefSelectProps } from 'antd';
import React, { Ref, RefObject, useEffect, useImperativeHandle, useState } from 'react';

const MODAL_WRAP_CLASSNAME = 'ant-modal-wrap';

const getModalContainer = () => {
  return document.getElementsByClassName(MODAL_WRAP_CLASSNAME)?.[0] as HTMLElement;
};

interface ModalEscapeHandlers {
  onBlur?: <T extends HTMLInputElement | HTMLTextAreaElement>(
    e: React.FocusEvent<T> | React.KeyboardEvent<T>,
    previousValue?: string,
    tagID?: number,
  ) => void;
}

interface ModalTextEscape extends ModalEscapeHandlers {
  inputRef?: React.RefObject<HTMLInputElement | InputRef>;
  onFocus: <T extends HTMLInputElement | HTMLTextAreaElement>(
    e: React.FocusEvent<T & EventTarget, Element>,
  ) => void;
}

interface ModalSelectEscape {
  inputRef?: React.RefObject<RefSelectProps>;
  onFocus: <T extends HTMLInputElement>(e: React.FocusEvent<T & EventTarget, Element>) => void;
  onBlur?: (
    e: React.FocusEvent<HTMLElement> | React.KeyboardEvent<HTMLElement>,
    previousValue?: string,
    tagID?: number,
  ) => void;
}

interface ModalNumberEscape extends ModalEscapeHandlers {
  onFocus: (e: React.FocusEvent<HTMLInputElement & EventTarget, Element>) => void;
  inputRef?: Ref<HTMLInputElement>;
}

const onEsc = (
  focused: boolean,
  inputRef: React.RefObject<HTMLInputElement | AntdInputRef | RefSelectProps>,
  event: KeyboardEvent,
  handleFocused: (focused: boolean) => void,
) => {
  if (focused && event.key === 'Escape') {
    event.stopPropagation();
    inputRef.current?.blur();
    getModalContainer().focus();
    handleFocused(false);
  }
};

const onClick = (
  blurred: boolean,
  focused: boolean,
  inputRef: React.RefObject<HTMLInputElement | AntdInputRef>,
  event: MouseEvent,
  handleFocused: (focused: boolean) => void,
) => {
  if (blurred && focused) {
    event.stopPropagation();
    handleFocused(false);
  } else if (focused && (event.target as HTMLElement).className === MODAL_WRAP_CLASSNAME) {
    event.stopPropagation();
    inputRef.current?.blur();
    handleFocused(false);
  }
};

const onClickSelect = (
  event: MouseEvent,
  isOpen: boolean,
  hasOpened: boolean,
  setHasOpened: (hasOpened: boolean) => void,
) => {
  if (isOpen && (event.target as HTMLElement).className === MODAL_WRAP_CLASSNAME) {
    event.stopPropagation();
  }

  if (hasOpened && !isOpen && (event.target as HTMLElement).className === MODAL_WRAP_CLASSNAME) {
    event.stopPropagation();
    event.preventDefault();
    setHasOpened(false);
  }
};

export const useModalNumberEscape = (
  ref: React.ForwardedRef<HTMLInputElement>,
  onBlur?: <HTMLInputElement>(
    e: React.FocusEvent<HTMLInputElement> | React.KeyboardEvent<HTMLInputElement>,
    previousValue?: string,
    tagID?: string,
  ) => void,
): ModalNumberEscape => {
  const inputRef = React.createRef<HTMLInputElement>();
  useImperativeHandle(ref, () => inputRef?.current as HTMLInputElement);

  const input = inputRef?.current;

  const [focused, setFocused] = useState(false);
  const [blurred, setBlurred] = useState(false);

  useEffect(() => {
    const handleEsc = (event: KeyboardEvent) => {
      if (focused && event.key === 'Escape') {
        onEsc(focused, inputRef, event, setFocused);
      }
    };

    const handleClick = (event: MouseEvent) => {
      onClick(blurred, focused, inputRef, event, setFocused);
    };

    input?.addEventListener('keydown', handleEsc);
    window.addEventListener('click', handleClick, true);
    return () => {
      input?.removeEventListener('keydown', handleEsc);
      window.removeEventListener('click', handleClick, true);
    };
  }, [blurred, focused, input, inputRef]);

  const handleBlur = <HTMLInputElement>(
    e: React.FocusEvent<HTMLInputElement> | React.KeyboardEvent<HTMLInputElement>,
    previousValue?: string,
  ) => {
    setBlurred(true);
    if (onBlur) onBlur(e, previousValue);
  };

  const onFocus = () => {
    setFocused(true);
    setBlurred(false);
  };
  return { inputRef, onBlur: handleBlur, onFocus };
};

export const useModalTextEscape = <T>(
  ref?: React.ForwardedRef<T>,
  onBlur?: <T extends HTMLInputElement | HTMLTextAreaElement>(
    e: React.FocusEvent<T> | React.KeyboardEvent<T>,
    previousValue?: string,
    tagID?: string,
  ) => void,
): ModalTextEscape => {
  const inputRef = React.createRef<AntdInputRef>();
  const [focused, setFocused] = useState(false);
  const [blurred, setBlurred] = useState(false);

  useImperativeHandle(ref, () => inputRef?.current as T);

  useEffect(() => {
    const handleEsc = (event: KeyboardEvent) => {
      if (focused && event.key === 'Escape') {
        onEsc(focused, inputRef, event, setFocused);
      }
    };
    const handleClick = (event: MouseEvent) => {
      onClick(blurred, focused, inputRef, event, setFocused);
    };
    document.addEventListener('keydown', handleEsc, true);
    document.addEventListener('click', handleClick, true);
    return () => {
      document.removeEventListener('keydown', handleEsc, true);
      document.removeEventListener('click', handleClick, true);
    };
  }, [focused, inputRef, blurred]);

  const handleBlur = <T extends HTMLInputElement | HTMLTextAreaElement>(
    e: React.FocusEvent<T> | React.KeyboardEvent<T>,
    previousValue?: string,
  ) => {
    setBlurred(true);
    if (onBlur) onBlur(e, previousValue);
  };

  const onFocus = () => {
    setFocused(true);
    setBlurred(false);
  };
  return { inputRef, onBlur: handleBlur, onFocus };
};

export const useModalSelectEscape = (
  containerRef: RefObject<HTMLDivElement>,
  isOpen: boolean,
  ref?: React.ForwardedRef<RefSelectProps>,
  onBlur?: (
    e: React.FocusEvent<HTMLElement> | React.KeyboardEvent<HTMLElement>,
    previousValue?: string,
    tagID?: string,
  ) => void,
): ModalSelectEscape => {
  const inputRef = React.createRef<RefSelectProps>();
  const [focused, setFocused] = useState(false);
  const [blurred, setBlurred] = useState(false);
  const [hasOpened, setHasOpened] = useState(false);

  useImperativeHandle(ref, () => inputRef.current as RefSelectProps);

  const input = containerRef.current;

  useEffect(() => {
    if (isOpen) {
      setHasOpened(true);
    }
  }, [isOpen]);

  useEffect(() => {
    const handleClick = (event: MouseEvent) => {
      onClickSelect(event, isOpen, hasOpened, setHasOpened);
    };

    const handleEsc = (event: KeyboardEvent) => {
      if (focused && event.key === 'Escape') {
        onEsc(focused, inputRef, event, setFocused);
      }
    };
    input?.addEventListener('keydown', handleEsc, true);
    window.addEventListener('click', handleClick, true);
    return () => {
      input?.addEventListener('keydown', handleEsc, true);
      window.removeEventListener('click', handleClick, true);
    };
  }, [blurred, focused, input, inputRef, containerRef, setFocused, isOpen, hasOpened]);

  const handleBlur = (
    e: React.FocusEvent<HTMLElement> | React.KeyboardEvent<HTMLElement>,
    previousValue?: string,
  ) => {
    setBlurred(true);
    if (onBlur) onBlur(e, previousValue);
  };

  const onFocus = () => {
    setFocused(true);
  };
  return { inputRef, onBlur: handleBlur, onFocus };
};
