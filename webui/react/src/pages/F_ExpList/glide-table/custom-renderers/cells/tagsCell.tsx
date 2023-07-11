import {
  CustomCell,
  CustomRenderer,
  getMiddleCenterBias,
  GridCellKind,
  measureTextCached,
  Rectangle,
} from '@hpe.com/glide-data-grid';

import { roundedRect } from '../utils';

interface TagsCellProps {
  readonly kind: 'tags-cell';
  readonly tags: readonly string[];
  readonly readonly?: boolean;
  readonly possibleTags: readonly {
    tag: string;
    color: string;
  }[];
}

export type TagsCell = CustomCell<TagsCellProps>;

const tagHeight = 20;
const innerPad = 6;

const renderer: CustomRenderer<TagsCell> = {
  draw: (args, cell) => {
    const { ctx, theme, rect } = args;
    const { possibleTags, tags } = cell.data;

    const drawArea: Rectangle = {
      height: rect.height - 2 * theme.cellVerticalPadding,
      width: rect.width - 2 * theme.cellHorizontalPadding,
      x: rect.x + theme.cellHorizontalPadding,
      y: rect.y + theme.cellVerticalPadding,
    };
    const rows = Math.max(1, Math.floor(drawArea.height / (tagHeight + innerPad)));

    let x = drawArea.x;
    let row = 1;
    let y = drawArea.y + (drawArea.height - rows * tagHeight - (rows - 1) * innerPad) / 2;
    for (const tag of tags) {
      const color = possibleTags.find((t) => t.tag === tag)?.color ?? theme.bgBubble;

      ctx.font = `12px ${theme.fontFamily}`;
      const metrics = measureTextCached(tag, ctx);
      const width = metrics.width + innerPad * 2;
      const textY = tagHeight / 2;

      if (x !== drawArea.x && x + width > drawArea.x + drawArea.width && row < rows) {
        row++;
        y += tagHeight + innerPad;
        x = drawArea.x;
      }

      ctx.fillStyle = color;
      ctx.lineWidth = 2;
      ctx.strokeStyle = theme.textBubble;
      ctx.beginPath();
      roundedRect(ctx, x, y, width, tagHeight, tagHeight / 2);
      ctx.stroke();
      ctx.fill();

      ctx.fillStyle = theme.textDark;
      ctx.fillText(
        tag,
        x + innerPad,
        y + textY + getMiddleCenterBias(ctx, `12px ${theme.fontFamily}`),
      );

      x += width + 8;
      if (x > drawArea.x + drawArea.width && row >= rows) break;
    }

    return true;
  },
  // eslint-disable-next-line
  isMatch: (c): c is TagsCell => (c.data as any).kind === 'tags-cell',
  kind: GridCellKind.Custom,
  onPaste: (v, d) => ({
    ...d,
    tags: d.possibleTags
      .map((x) => x.tag)
      .filter((x) =>
        v
          .split(',')
          .map((s) => s.trim())
          .includes(x),
      ),
  }),
  provideEditor: () => undefined,
};

export default renderer;
