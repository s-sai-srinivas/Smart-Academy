import { type ChangeEvent } from 'react';
import { LANGUAGE_MAP } from './codeEditorConstants';

export interface LanguageSelectorProps {
  value: number;
  onChange: (value: number) => void;
}

function LanguageSelector({ value, onChange }: LanguageSelectorProps) {
  return (
    <select
      className="language-select"
      value={value}
      onChange={(e: ChangeEvent<HTMLSelectElement>) => onChange(parseInt(e.target.value))}
    >
      {Object.entries(LANGUAGE_MAP).map(([id, { name }]) => (
        <option key={id} value={id}>
          {name}
        </option>
      ))}
    </select>
  );
}

export default LanguageSelector;
