import { Radio } from 'antd';
import { RadioChangeEvent } from 'antd/lib/radio';
import Icon, { IconName, IconSize } from 'determined-ui/Icon';
import Tooltip from 'determined-ui/Tooltip';
import React, {
  RefObject,
  useCallback,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import { throttle } from 'throttle-debounce';

import css from './RadioGroup.module.scss';

interface ResizeInfo {
  height: number;
  width: number;
  x: number;
  y: number;
}

const defaultResizeInfo = {
  height: 0,
  width: 0,
  x: 0,
  y: 0,
};

export const DEFAULT_RESIZE_THROTTLE_TIME = 500;

const useResize = (ref?: RefObject<HTMLElement>): ResizeInfo => {
  const [resizeInfo, setResizeInfo] = useState<ResizeInfo>(defaultResizeInfo);

  useLayoutEffect(() => {
    let element = document.body;
    if (ref) {
      if (ref.current) element = ref.current;
      else return;
    }

    const handleResize: ResizeObserverCallback = (entries: ResizeObserverEntry[]) => {
      // Check to make sure the ref container is being observed for resize.
      const elements = entries.map((entry: ResizeObserverEntry) => entry.target);
      if (!element || elements.indexOf(element) === -1) return;

      const rect = element.getBoundingClientRect();
      setResizeInfo(rect);
    };
    const resizeObserver = new ResizeObserver(handleResize);
    resizeObserver.observe(element);

    // Set initial resize info
    const rect = element.getBoundingClientRect();
    setResizeInfo(rect);

    return (): void => resizeObserver.unobserve(element);
  }, [ref]);

  return resizeInfo;
};

interface Props {
  defaultValue?: string;
  iconOnly?: boolean;
  onChange?: (id: string) => void;
  options: RadioGroupOption[];
  value?: string;
  radioType?: 'button' | 'radio';
}

export interface RadioGroupOption {
  icon?: IconName;
  iconSize?: IconSize;
  id: string;
  label: string;
}

interface SizeInfo {
  baseHeight: number;
  baseWidth: number;
  parentWidth: number;
}

const RESIZE_THROTTLE_TIME = 500;
const PARENT_WIDTH_BUFFER = 16;
const HEIGHT_LIMIT = 50;

const RadioGroup: React.FC<Props> = ({
  defaultValue,
  iconOnly = false,
  onChange,
  options,
  value,
  radioType = 'button',
}: Props) => {
  const baseRef = useRef<HTMLDivElement>(null);
  const originalWidth = useRef<number>();
  const resize = useResize();
  const [showLabels, setShowLabels] = useState(true);
  const [sizes, setSizes] = useState<SizeInfo>({ baseHeight: 0, baseWidth: 0, parentWidth: 0 });
  const classes = [css.base];

  const hasIconsAndLabels = useMemo(() => {
    if (options.length === 0) return false;
    return options.reduce((acc, option) => acc || (!!option.icon && !!option.label), false);
  }, [options]);

  if (iconOnly) classes.push(css.iconOnly);

  const handleChange = useCallback(
    (e: RadioChangeEvent) => {
      if (onChange) onChange(e.target.value);
    },
    [onChange],
  );

  /*
   * Dynamically check to see if labels can be shown along with the icons,
   * if there's room to show both and both are available.
   */
  useEffect(() => {
    if (!hasIconsAndLabels || !baseRef.current) return;
    if (sizes.baseWidth === 0 || sizes.parentWidth === 0) return;

    setShowLabels((prev) => {
      if (!originalWidth.current) return prev;
      if (prev && sizes.baseHeight > HEIGHT_LIMIT) return false;
      if (!prev && originalWidth.current < sizes.parentWidth) return true;
      return prev;
    });
  }, [hasIconsAndLabels, sizes]);

  /*
   * Update parent and component sizes upon resize of the window,
   * at a throttled rate.
   */
  useEffect(() => {
    const throttleFunc = throttle(RESIZE_THROTTLE_TIME, () => {
      if (!hasIconsAndLabels || !baseRef.current) return;
      const parent = baseRef.current.parentElement;
      if (!parent) return;

      const parentRect = parent.getBoundingClientRect();
      if (!parentRect) return;

      const baseRect = baseRef.current.getBoundingClientRect();
      if (!originalWidth.current) originalWidth.current = baseRect.width;

      setSizes({
        baseHeight: baseRect.height,
        baseWidth: baseRect.width,
        parentWidth: parentRect.width - PARENT_WIDTH_BUFFER,
      });
    });

    throttleFunc();
  }, [hasIconsAndLabels, resize]);

  return (
    <Radio.Group
      className={classes.join(' ')}
      defaultValue={defaultValue}
      ref={baseRef}
      value={value}
      onChange={handleChange}>
      {options.map((option) =>
        !showLabels || iconOnly ? (
          <Tooltip content={option.label} key={option.id} placement="top">
            {radioType === 'radio' ? (
              <Radio className={css.option} value={option.id}>
                {option.icon && <Icon decorative name={option.icon} size={option.iconSize} />}
                {option.label && showLabels && !iconOnly && (
                  <span className={css.label}>{option.label}</span>
                )}
              </Radio>
            ) : (
              <Radio.Button className={css.option} value={option.id}>
                {option.icon && <Icon decorative name={option.icon} size={option.iconSize} />}
                {option.label && showLabels && !iconOnly && (
                  <span className={css.label}>{option.label}</span>
                )}
              </Radio.Button>
            )}
          </Tooltip>
        ) : radioType === 'radio' ? (
          <Radio className={css.option} key={option.id} value={option.id}>
            {option.icon && <Icon decorative name={option.icon} size={option.iconSize} />}
            {option.label && showLabels && !iconOnly && (
              <span className={css.label}>{option.label}</span>
            )}
          </Radio>
        ) : (
          <Radio.Button className={css.option} key={option.id} value={option.id}>
            {option.icon && <Icon decorative name={option.icon} size={option.iconSize} />}
            {option.label && showLabels && !iconOnly && (
              <span className={css.label}>{option.label}</span>
            )}
          </Radio.Button>
        ),
      )}
    </Radio.Group>
  );
};

export default RadioGroup;
