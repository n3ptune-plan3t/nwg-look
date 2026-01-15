#!/usr/bin/env python3
"""
GTK Theme Color Extractor and Template Manager
Extracts colors from GTK themes and applies them to application configs
"""

import os
import re
import json
import shutil
from pathlib import Path
from typing import Dict, List, Optional
import configparser


class GTKColorExtractor:
    """Extract colors from GTK theme CSS files"""
    
    def __init__(self):
        self.theme_paths = [
            Path.home() / ".themes",
            Path.home() / ".local/share/themes",
            Path("/usr/share/themes")
        ]
        
    def find_theme_path(self, theme_name: str) -> Optional[Path]:
        """Find the path to a GTK theme"""
        for base_path in self.theme_paths:
            theme_path = base_path / theme_name / "gtk-3.0"
            if theme_path.exists():
                return theme_path
        return None
    
    def extract_colors(self, theme_name: str) -> Dict[str, str]:
        """Extract color palette from GTK theme"""
        theme_path = self.find_theme_path(theme_name)
        if not theme_path:
            print(f"Theme {theme_name} not found")
            return {}
        
        css_file = theme_path / "gtk.css"
        if not css_file.exists():
            print(f"gtk.css not found in {theme_path}")
            return {}
        
        colors = {}
        
        with open(css_file, 'r', encoding='utf-8', errors='ignore') as f:
            content = f.read()
            
            # Extract @define-color declarations
            color_pattern = r'@define-color\s+(\w+)\s+([#\w(),.\s]+);'
            matches = re.findall(color_pattern, content)
            
            for name, value in matches:
                # Clean up the value
                value = value.strip()
                # Store the color
                colors[name] = value
            
            # Also look for common CSS color variables
            var_pattern = r'--(\w+-\w+(?:-\w+)*)\s*:\s*([#\w(),.\s]+);'
            var_matches = re.findall(var_pattern, content)
            
            for name, value in var_matches:
                value = value.strip()
                colors[name] = value
        
        # Resolve color references
        colors = self._resolve_color_references(colors)
        
        # Generate standard color palette
        palette = self._generate_standard_palette(colors)
        
        return palette
    
    def _resolve_color_references(self, colors: Dict[str, str]) -> Dict[str, str]:
        """Resolve color references like @theme_bg_color"""
        resolved = {}
        max_iterations = 10
        
        for _ in range(max_iterations):
            changed = False
            for name, value in colors.items():
                # Check if value references another color
                if '@' in value:
                    ref_match = re.search(r'@(\w+)', value)
                    if ref_match:
                        ref_name = ref_match.group(1)
                        if ref_name in colors:
                            # Replace reference with actual value
                            new_value = value.replace(f'@{ref_name}', colors[ref_name])
                            if new_value != value:
                                colors[name] = new_value
                                changed = True
                        else:
                            resolved[name] = value
                    else:
                        resolved[name] = value
                else:
                    resolved[name] = value
            
            if not changed:
                break
        
        return resolved
    
    def _generate_standard_palette(self, colors: Dict[str, str]) -> Dict[str, str]:
        """Generate a standardized color palette"""
        palette = {
            'background': '#1e1e1e',
            'foreground': '#d4d4d4',
            'cursor': '#d4d4d4',
            'color0': '#000000',
            'color1': '#cd3131',
            'color2': '#0dbc79',
            'color3': '#e5e510',
            'color4': '#2472c8',
            'color5': '#bc3fbc',
            'color6': '#11a8cd',
            'color7': '#e5e5e5',
            'color8': '#666666',
            'color9': '#f14c4c',
            'color10': '#23d18b',
            'color11': '#f5f543',
            'color12': '#3b8eea',
            'color13': '#d670d6',
            'color14': '#29b8db',
            'color15': '#e5e5e5',
        }
        
        # Map GTK theme colors to standard palette
        color_mapping = {
            'theme_bg_color': 'background',
            'theme_fg_color': 'foreground',
            'theme_base_color': 'background',
            'theme_text_color': 'foreground',
            'theme_selected_bg_color': 'color4',
            'theme_selected_fg_color': 'foreground',
            'warning_color': 'color3',
            'error_color': 'color1',
            'success_color': 'color2',
        }
        
        for gtk_name, std_name in color_mapping.items():
            if gtk_name in colors:
                palette[std_name] = self._normalize_color(colors[gtk_name])
        
        return palette


    def _normalize_color(self, color: str) -> str:
        """Normalize color to hex format"""
        color = color.strip()
        
        # Already hex
        if color.startswith('#'):
            return color
        
        # RGB/RGBA format
        if 'rgb' in color.lower():
            match = re.search(r'rgba?\((\d+),\s*(\d+),\s*(\d+)', color)
            if match:
                r, g, b = match.groups()
                return f'#{int(r):02x}{int(g):02x}{int(b):02x}'
        
        return color


class TemplateManager:
    """Manage color templates for various applications"""
    
    def __init__(self):
        self.config_dir = Path.home() / ".config/nwg-look/templates"
        self.config_dir.mkdir(parents=True, exist_ok=True)
        self._create_default_templates()
    
    def _create_default_templates(self):
        """Create default templates if they don't exist"""
        templates = {
            'alacritty.yml': self._alacritty_template(),
            'waybar-colors.css': self._waybar_template(),
            'kitty.conf': self._kitty_template(),
            'rofi-colors.rasi': self._rofi_template(),
            'dunst.conf': self._dunst_template(),
        }
        
        for filename, content in templates.items():
            template_file = self.config_dir / filename
            if not template_file.exists():
                with open(template_file, 'w') as f:
                    f.write(content)
    
    def _alacritty_template(self) -> str:
        return '''# Alacritty colors - Generated by nwg-look
colors:
  primary:
    background: '{background}'
    foreground: '{foreground}'
  cursor:
    text: '{background}'
    cursor: '{cursor}'
  normal:
    black:   '{color0}'
    red:     '{color1}'
    green:   '{color2}'
    yellow:  '{color3}'
    blue:    '{color4}'
    magenta: '{color5}'
    cyan:    '{color6}'
    white:   '{color7}'
  bright:
    black:   '{color8}'
    red:     '{color9}'
    green:   '{color10}'
    yellow:  '{color11}'
    blue:    '{color12}'
    magenta: '{color13}'
    cyan:    '{color14}'
    white:   '{color15}'
'''
    
    def _waybar_template(self) -> str:
        return '''/* Waybar colors - Generated by nwg-look */
@define-color background {background};
@define-color foreground {foreground};
@define-color color0 {color0};
@define-color color1 {color1};
@define-color color2 {color2};
@define-color color3 {color3};
@define-color color4 {color4};
@define-color color5 {color5};
@define-color color6 {color6};
@define-color color7 {color7};
@define-color color8 {color8};

window#waybar {{
    background-color: @background;
    color: @foreground;
}}
'''
    
    def _kitty_template(self) -> str:
        return '''# Kitty colors - Generated by nwg-look
foreground {foreground}
background {background}
cursor {cursor}

color0 {color0}
color1 {color1}
color2 {color2}
color3 {color3}
color4 {color4}
color5 {color5}
color6 {color6}
color7 {color7}
color8 {color8}
color9 {color9}
color10 {color10}
color11 {color11}
color12 {color12}
color13 {color13}
color14 {color14}
color15 {color15}
'''
    
    def _rofi_template(self) -> str:
        return '''/* Rofi colors - Generated by nwg-look */
* {{
    background: {background};
    foreground: {foreground};
    selected: {color4};
    active: {color2};
    urgent: {color1};
}}
'''
    
    def _dunst_template(self) -> str:
        return '''# Dunst colors - Generated by nwg-look
[global]
    background = "{background}"
    foreground = "{foreground}"
    
[urgency_low]
    background = "{background}"
    foreground = "{foreground}"

[urgency_normal]
    background = "{color4}"
    foreground = "{foreground}"

[urgency_critical]
    background = "{color1}"
    foreground = "{foreground}"
'''
    
    def apply_colors(self, colors: Dict[str, str]):
        """Apply colors to all templates and copy to config locations"""
        destinations = {
            'alacritty.yml': Path.home() / '.config/alacritty/colors.yml',
            'waybar-colors.css': Path.home() / '.config/waybar/colors.css',
            'kitty.conf': Path.home() / '.config/kitty/theme.conf',
            'rofi-colors.rasi': Path.home() / '.config/rofi/colors.rasi',
            'dunst.conf': Path.home() / '.config/dunst/dunstrc-colors',
        }
        
        for template_name, dest_path in destinations.items():
            template_path = self.config_dir / template_name
            
            if not template_path.exists():
                continue
            
            # Read template
            with open(template_path, 'r') as f:
                template = f.read()
            
            # Apply colors
            output = template.format(**colors)
            
            # Create destination directory if needed
            dest_path.parent.mkdir(parents=True, exist_ok=True)
            
            # Write to destination
            with open(dest_path, 'w') as f:
                f.write(output)
            
            print(f"✓ Applied colors to {dest_path}")


class ColorSchemeManager:
    """Main manager for color scheme application"""
    
    def __init__(self):
        self.extractor = GTKColorExtractor()
        self.template_mgr = TemplateManager()
        self.config_file = Path.home() / '.config/nwg-look/color-scheme.json'
        self.config_file.parent.mkdir(parents=True, exist_ok=True)
    
    def apply_theme(self, theme_name: str):
        """Extract colors from GTK theme and apply to all configs"""
        print(f"Extracting colors from GTK theme: {theme_name}")
        
        colors = self.extractor.extract_colors(theme_name)
        
        if not colors:
            print("Failed to extract colors from theme")
            return False
        
        print(f"Extracted {len(colors)} colors")
        
        # Save color scheme
        self._save_color_scheme(theme_name, colors)
        
        # Apply to templates
        print("\nApplying colors to application configs...")
        self.template_mgr.apply_colors(colors)
        
        print(f"\n✓ Successfully applied {theme_name} colors!")
        return True
    
    def _save_color_scheme(self, theme_name: str, colors: Dict[str, str]):
        """Save current color scheme to config"""
        config = {
            'theme': theme_name,
            'colors': colors,
            'applied_at': str(Path.ctime(Path.home()))
        }
        
        with open(self.config_file, 'w') as f:
            json.dump(config, f, indent=2)
    
    def get_current_scheme(self) -> Optional[Dict]:
        """Get currently applied color scheme"""
        if not self.config_file.exists():
            return None
        
        with open(self.config_file, 'r') as f:
            return json.load(f)


def main():
    """CLI interface for testing"""
    import sys
    
    if len(sys.argv) < 2:
        print("Usage: python gtk_color_extractor.py <theme_name>")
        print("Example: python gtk_color_extractor.py Adwaita-dark")
        sys.exit(1)
    
    theme_name = sys.argv[1]
    
    manager = ColorSchemeManager()
    manager.apply_theme(theme_name)


if __name__ == '__main__':
    main()
