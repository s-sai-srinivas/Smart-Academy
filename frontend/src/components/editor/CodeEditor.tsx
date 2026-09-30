import Editor from '@monaco-editor/react';
import { useTheme } from '../../context/ThemeContext';
import { DEFAULT_CODE, LANGUAGE_MAP } from './codeEditorConstants';

export interface CodeEditorProps {
  languageId: number;
  code?: string;
  onChange?: (value: string) => void;
  onEscape?: () => void;
  readOnly?: boolean;
}

// Extend Window interface for Monaco global
type MonacoType = typeof import('monaco-editor');

declare global {
  interface Window {
    __monaco?: MonacoType;
  }
}

function CodeEditor({ languageId, code, onChange, onEscape, readOnly = false }: CodeEditorProps) {
  const language = LANGUAGE_MAP[languageId]?.monacoLang || 'python';
  const { theme } = useTheme();
  const editorTheme = theme === 'dark' ? 'vs-dark' : 'light';

  return (
    <div className="code-editor" style={{ height: '100%' }}>
      <Editor
        height="100%"
        language={language}
        value={code || DEFAULT_CODE[languageId] || ''}
        onChange={(value) => onChange?.(value ?? '')}
        theme={editorTheme}
        beforeMount={(monaco) => {
          window.__monaco = monaco;
        }}
        onMount={(editor) => {
          const monaco = window.__monaco;
          if (!monaco) return;
          editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyV, () => {});
          editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyX, () => {});
          editor.addCommand(monaco.KeyMod.Shift | monaco.KeyCode.Insert, () => {});
          const domNode = editor.getDomNode();
          if (domNode) {
            domNode.addEventListener(
              'paste',
              (e: Event) => {
                e.preventDefault();
                e.stopPropagation();
              },
              true
            );
            domNode.addEventListener(
              'drop',
              (e: Event) => {
                e.preventDefault();
                e.stopPropagation();
              },
              true
            );
          }
          editor.addCommand(monaco.KeyCode.ContextMenu, () => {});
          if (onEscape) {
            editor.addCommand(monaco.KeyCode.Escape, () => {
              onEscape();
            });
          }
        }}
        options={{
          minimap: { enabled: false },
          fontSize: 14,
          scrollBeyondLastLine: false,
          automaticLayout: true,
          padding: { top: 10 },
          contextmenu: false,
          copyWithSyntaxHighlighting: false,
          readOnly,
        }}
      />
    </div>
  );
}

export default CodeEditor;
