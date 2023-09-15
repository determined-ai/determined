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

  // Only one overlay will exist when a menu is open
  // the overlay that we want will be the first item
  // in the list.
  const overlay = overlays?.[0] as HTMLElement;

  const menuBody =
    overlay?.className === MODAL_WRAP_CLASSNAME
      ? overlay
      : (document.getElementsByClassName(DRAWER_BODY_CLASSNAME)?.[0] as HTMLElement);

  return {
    focusMenu: () => menuBody?.focus(),
    overlayClassname: overlay?.className,
    // If an overlay does not exist then the input is not
    // inside of a menu or modal and we do not want to
    // alter any event behavior
    overlayExists: !!overlay,
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
  const { overlayExists } = getOverlayAndMenuBodyElement();
  if (!overlayExists) return;
  if (focused && event.key === 'Escape') {
    event.stopPropagation();
    inputRef.current?.blur();
    getOverlayAndMenuBodyElement()?.focusMenu();
    handleFocused(false);
  }
};

const onInputNumberClick = (
  blurred: boolean,
  focused: boolean,
  inputRef: React.RefObject<HTMLInputElement>,
  event: MouseEvent,
  handleFocused: (focused: boolean) => void,
) => {
  const { overlayClassname, overlayExists } = getOverlayAndMenuBodyElement();
  if (!overlayExists) return;
  const overlayClicked = (event.target as HTMLElement).className === overlayClassname;

  if (focused && (overlayClicked || event.target !== inputRef.current)) {
    event.stopPropagation();
    handleFocused(false);
    // If the input is not already blurred then perform the blur
    if (!blurred) inputRef.current?.blur();
  }
};

const onInputClick = (
  blurred: boolean,
  focused: boolean,
  inputRef: React.RefObject<AntdInputRef>,
  event: MouseEvent,
  handleFocused: (focused: boolean) => void,
) => {
  const { overlayClassname, overlayExists } = getOverlayAndMenuBodyElement();
  if (!overlayExists) return;
  const overlayClicked = (event.target as HTMLElement).className === overlayClassname;

  if (focused && (overlayClicked || event.target !== inputRef.current?.input)) {
    event.stopPropagation();
    handleFocused(false);
    // If the input is not already blurred then perform the blur
    if (!blurred) inputRef.current?.blur();
  }
};

const onClickSelect = (
  blurred: boolean,
  event: MouseEvent,
  isOpen: boolean,
  hasOpened: boolean,
  setHasOpened: (hasOpened: boolean) => void,
) => {
  const targetClassname = (event.target as HTMLElement).className;
  const { overlayClassname, overlayExists } = getOverlayAndMenuBodyElement();
  if (!overlayExists) return;
  if (isOpen && targetClassname === overlayClassname) {
    event.stopPropagation();
  }

  if (hasOpened && !isOpen && targetClassname === overlayClassname) {
    event.stopPropagation();
    // If hasOpened is true in this instance
    // then the event above will close the
    // currently open Modal or Menu so we must stop the
    // event.
    setHasOpened(false);
  }

  // If any element besides the overlay has been clicked
  // set hasOpened as false to signify that the
  // dropdown is now closed
  if (blurred && hasOpened && targetClassname !== overlayClassname) setHasOpened(false);
};

const onEscSelect = (
  focused: boolean,
  inputRef: React.RefObject<HTMLInputElement | AntdInputRef | RefSelectProps>,
  event: KeyboardEvent,
  handleFocused: (focused: boolean) => void,
  setHasOpened: (hasOpened: boolean) => void,
) => {
  const { overlayExists, focusMenu } = getOverlayAndMenuBodyElement();
  if (!overlayExists) return;
  if (focused && event.key === 'Escape') {
    event.stopPropagation();
    inputRef.current?.blur();
    focusMenu();
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

  const [focused, setFocused] = useState(false);
  const [blurred, setBlurred] = useState(false);

  useEffect(() => {
    const handleEsc = (event: KeyboardEvent) => {
      onEsc(focused, inputRef, event, setFocused);
    };

    const handleClick = (event: MouseEvent) => {
      onInputNumberClick(blurred, focused, inputRef, event, setFocused);
    };

    window.addEventListener('keydown', handleEsc, true);
    window.addEventListener('click', handleClick, true);
    return () => {
      window.removeEventListener('keydown', handleEsc, true);
      window.removeEventListener('click', handleClick, true);
    };
  }, [blurred, focused]);

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
      onInputClick(blurred, focused, inputRef, event, setFocused);
    };
    document.addEventListener('keydown', handleEsc, true);
    document.addEventListener('click', handleClick, true);
    return () => {
      document.removeEventListener('keydown', handleEsc, true);
      document.removeEventListener('click', handleClick, true);
    };
  }, [focused, blurred]);

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
    /**
    By the time a click event is captured in an
    event handler the Select will have otherwise already
    become unfocused and isOpen will be false.
    hasOpened is used in place of "isOpen" to check
    if the select is still open at the time of the click
    event.
     */
    if (isOpen) setHasOpened(true);
  }, [isOpen]);

  useEffect(() => {
    const handleClick = (event: MouseEvent) => {
      onClickSelect(blurred, event, isOpen, hasOpened, setHasOpened);
    };

    const handleEsc = (event: KeyboardEvent) => {
      onEscSelect(focused, inputRef, event, setFocused, setHasOpened);
    };

    /**
     *  The Select component is a special case where
     *  a seperate container is needed in order to
     *  properly handle blurring the component.
     *  In this implementation the event handler is attached to the
     *  containerRef, when the Select is unfocused
     *  the containerRef is unmounted which allows
     *  the "Escape" events to propogate
     *  normally.
     */

    containerRef.current?.addEventListener('keydown', handleEsc);
    window.addEventListener('click', handleClick, true);
    return () => {
      // eslint-disable-next-line react-hooks/exhaustive-deps
      containerRef.current?.removeEventListener('keydown', handleEsc);

      window.removeEventListener('click', handleClick, true);
    };
  }, [blurred, containerRef, focused, setFocused, isOpen, hasOpened]);

  const handleBlur = (
    e: React.FocusEvent<HTMLElement> | React.KeyboardEvent<HTMLElement>,
    previousValue?: string,
  ) => {
    setBlurred(true);
    setFocused(false);
    onBlur?.(e, previousValue);
  };

  const onFocus = () => {
    setFocused(true);
  };

  return { inputRef, onBlur: handleBlur, onFocus };
};
