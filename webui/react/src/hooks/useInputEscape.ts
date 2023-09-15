import { InputRef as AntdInputRef, InputRef, RefSelectProps } from 'antd';
import React, { Ref, RefObject, useEffect, useImperativeHandle, useRef, useState } from 'react';

const DRAWER_BODY_CLASSNAME = 'ant-drawer-open';
const DRAWER_MASK_CLASSNAME = 'ant-drawer-mask';
const MODAL_WRAP_CLASSNAME = 'ant-modal-wrap';

const getOverlayAndMenuBodyElement = () => {
  // In order for the Escape button to be able to
  // close a menu after an input is unfocused
  // we need to be able to re-focus the window
  // to the body of the current menu.

  const overlays = [
    ...document.getElementsByClassName(MODAL_WRAP_CLASSNAME),
    ...document.getElementsByClassName(DRAWER_MASK_CLASSNAME),
  ];

  const overlay = overlays?.[0] as HTMLElement;

  const menuBody =
    overlay?.className === MODAL_WRAP_CLASSNAME
      ? overlay
      : (document.getElementsByClassName(DRAWER_BODY_CLASSNAME)?.[0] as HTMLElement);

  return {
    focusMenu: () => menuBody.focus(),
    overlayClassname: overlay?.className,
  };
};

interface InputEscape {
  onBlur?: <T extends HTMLInputElement | HTMLTextAreaElement>(
    e: React.FocusEvent<T> | React.KeyboardEvent<T>,
    previousValue?: string,
    tagID?: number,
  ) => void;
}

interface InputTextEscape extends InputEscape {
  inputRef?: React.RefObject<HTMLInputElement | InputRef>;
  onFocus: <T extends HTMLInputElement | HTMLTextAreaElement>(
    e: React.FocusEvent<T & EventTarget, Element>,
  ) => void;
}

interface InputNumberEscape extends InputEscape {
  onFocus: (e: React.FocusEvent<HTMLInputElement & EventTarget, Element>) => void;
  inputRef?: Ref<HTMLInputElement>;
}

interface SelectEscape {
  inputRef?: React.RefObject<RefSelectProps>;
  onFocus: <T extends HTMLInputElement>(e: React.FocusEvent<T & EventTarget, Element>) => void;
  onBlur?: (
    e: React.FocusEvent<HTMLElement> | React.KeyboardEvent<HTMLElement>,
    previousValue?: string,
    tagID?: number,
  ) => void;
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
    getOverlayAndMenuBodyElement()?.focusMenu();
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
  const overlayClicked =
    (event.target as HTMLElement).className === getOverlayAndMenuBodyElement()?.overlayClassname;

  if (blurred && focused && overlayClicked) {
    event.stopPropagation();
    handleFocused(false);
  } else if (focused && overlayClicked) {
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
  const isTargetOverlay =
    getOverlayAndMenuBodyElement()?.overlayClassname === (event.target as HTMLElement).className;
  if (isTargetOverlay && (isOpen || hasOpened)) {
    event.stopPropagation();
    if (!isOpen) setHasOpened(false);
  }
};

const onEscSelect = (
  focused: boolean,
  inputRef: React.RefObject<HTMLInputElement | AntdInputRef | RefSelectProps>,
  event: KeyboardEvent,
  handleFocused: (focused: boolean) => void,
  setHasOpened: (hasOpened: boolean) => void,
) => {
  if (focused && event.key === 'Escape') {
    event.stopPropagation();
    inputRef.current?.blur();
    getOverlayAndMenuBodyElement()?.focusMenu();
    handleFocused(false);
    setHasOpened(false);
  }
};

export const useInputNumberEscape = (
  ref: React.ForwardedRef<HTMLInputElement>,
  onBlur?: <HTMLInputElement>(
    e: React.FocusEvent<HTMLInputElement> | React.KeyboardEvent<HTMLInputElement>,
    previousValue?: string,
    tagID?: string,
  ) => void,
): InputNumberEscape => {
  const inputRef = useRef<HTMLInputElement>(null);
  useImperativeHandle(ref, () => inputRef?.current as HTMLInputElement);

  const input = inputRef?.current;

  const [focused, setFocused] = useState(false);
  const [blurred, setBlurred] = useState(false);

  useEffect(() => {
    const handleEsc = (event: KeyboardEvent) => {
      onEsc(focused, inputRef, event, setFocused);
    };

    const handleClick = (event: MouseEvent) => {
      onClick(blurred, focused, inputRef, event, setFocused);
    };

    window.addEventListener('keydown', handleEsc, true);
    window.addEventListener('click', handleClick, true);
    return () => {
      window.removeEventListener('keydown', handleEsc, true);
      window.removeEventListener('click', handleClick, true);
    };
  }, [blurred, focused, input, inputRef]);

  const handleBlur = <HTMLInputElement>(
    e: React.FocusEvent<HTMLInputElement> | React.KeyboardEvent<HTMLInputElement>,
    previousValue?: string,
  ) => {
    setBlurred(true);
    onBlur?.(e, previousValue);
  };

  const onFocus = () => {
    setFocused(true);
    setBlurred(false);
  };
  return { inputRef, onBlur: handleBlur, onFocus };
};

export const useInputEscape = <T>(
  ref?: React.ForwardedRef<T>,
  onBlur?: <T extends HTMLInputElement | HTMLTextAreaElement>(
    e: React.FocusEvent<T> | React.KeyboardEvent<T>,
    previousValue?: string,
    tagID?: string,
  ) => void,
): InputTextEscape => {
  const inputRef = useRef<AntdInputRef>(null);
  const [focused, setFocused] = useState(false);
  const [blurred, setBlurred] = useState(false);

  useImperativeHandle(ref, () => inputRef?.current as T);

  useEffect(() => {
    const handleEsc = (event: KeyboardEvent) => {
      onEsc(focused, inputRef, event, setFocused);
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
    onBlur?.(e, previousValue);
  };

  const onFocus = () => {
    setFocused(true);
    setBlurred(false);
  };
  return { inputRef, onBlur: handleBlur, onFocus };
};

export const useSelectEscape = (
  containerRef: RefObject<HTMLDivElement>,
  isOpen: boolean,
  ref?: React.ForwardedRef<RefSelectProps>,
  onBlur?: (
    e: React.FocusEvent<HTMLElement> | React.KeyboardEvent<HTMLElement>,
    previousValue?: string,
    tagID?: string,
  ) => void,
): SelectEscape => {
  const inputRef = useRef<RefSelectProps>(null);
  const [focused, setFocused] = useState(false);
  const [blurred, setBlurred] = useState(false);
  const [hasOpened, setHasOpened] = useState(false);

  useImperativeHandle(ref, () => inputRef.current as RefSelectProps);

  useEffect(() => {
    // By the time a click event is captured in an
    // event handler the Select will have otherwise already
    // become unfocused and isOpen will be false.
    // hasOpened is used in place of "isOpen" to check
    // if the select is still open at the time of the click
    // event.
    if (isOpen) setHasOpened(true);
  }, [isOpen]);

  useEffect(() => {
    const handleClick = (event: MouseEvent) => {
      onClickSelect(event, isOpen, hasOpened, setHasOpened);
    };

    const handleEsc = (event: KeyboardEvent) => {
      onEscSelect(focused, inputRef, event, setFocused, setHasOpened);
    };

    containerRef.current?.addEventListener('keydown', handleEsc);
    window.addEventListener('click', handleClick, true);
    return () => {
      // eslint-disable-next-line react-hooks/exhaustive-deps
      containerRef.current?.removeEventListener('keydown', handleEsc);

      window.removeEventListener('click', handleClick, true);
    };
  }, [blurred, containerRef, focused, inputRef, setFocused, isOpen, hasOpened]);

  const handleBlur = (
    e: React.FocusEvent<HTMLElement> | React.KeyboardEvent<HTMLElement>,
    previousValue?: string,
  ) => {
    setBlurred(true);
    onBlur?.(e, previousValue);
  };

  const onFocus = () => {
    setFocused(true);
  };

  return { inputRef, onBlur: handleBlur, onFocus };
};
