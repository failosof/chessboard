import os
import sys
import subprocess


def convert_svgs_to_pngs(directory):
    # Ensure the directory exists
    if not os.path.isdir(directory):
        print(f"Error: The specified directory '{directory}' does not exist.")
        sys.exit(1)

    # Process all SVG files in the directory
    for file_name in os.listdir(directory):
        if file_name.endswith(".svg"):
            input_path = os.path.join(directory, file_name)
            output_path = os.path.join(directory, file_name.replace(".svg", ".png"))

            # Convert SVG to PNG using Inkscape
            subprocess.run([
                "inkscape",
                input_path,
                "--export-type=png",
                "--export-filename", output_path
            ])
            print(f"Converted {file_name} to {output_path}")


if __name__ == "__main__":
    if len(sys.argv) != 2:
        print("Usage: python convert_svgs.py <directory>")
        sys.exit(1)

    input_output_dir = sys.argv[1]
    convert_svgs_to_pngs(input_output_dir)
