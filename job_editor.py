import json
import subprocess
import os
import sys
import tkinter as tk
from tkinter import ttk

# Global Variables
active_editor = []
main_tree = False
worker_config = {}
ui_config = {}
root = {}
columns = {}
rows = {}


# Function for editing cell
def edit_cell(event):
    
    global active_editor, main_tree

    if main_tree.identify_region(event.x, event.y) == "cell":

        column = main_tree.identify_column(event.x)
        row_id = main_tree.identify_row(event.y)

        # Get current cell value
        current_value = main_tree.item(row_id, 'values')[int(column[1:]) - 1]

        # Get cell bounding box for placement
        x, y, width, height = main_tree.bbox(row_id, column)

        # Create and place entry widget
        entry = tk.Entry(main_tree, width=width // 8)
        entry.place(x=x, y=y, width=width, height=height)
        entry.insert(0, current_value)
        entry.focus_set()

        def save_edit(event=None):
            global active_editor, main_tree
            new_value = entry.get()
            main_tree.set(row_id, column, new_value)
            entry.destroy()
            active_editor = []

        # Mark this cell as active so we can close it when other button pressed
        active_editor = [entry]
        
        entry.bind("<Return>", save_edit)
        entry.bind("<FocusOut>", save_edit)


# Function for saving all the rows data
def save_rows():
    global main_tree, worker_config, active_editor

    if len(active_editor) != 0:
        active_editor[0].event_generate("<FocusOut>")

    main_tree.focus_set()

    data = []
    for item_id in main_tree.get_children():
        values = main_tree.item(item_id, 'values')

        # Refusing to save empty row
        if all(not x for x in values):
            continue


        item_data = {
            'email': values[0], 
            'source_coin': values[1], 
            'target_coin': values[2], 
            'source_value': values[3], 
            'target_value': values[4], 
            'comparison': values[5], 
            'email_sent_count': values[6] 
        }

        data.append(item_data)

    worker_config['jobs'] = data

    with open('config.json', 'w') as f:
        json.dump(worker_config, f, indent=4)


# Function for deleting row(s)
def delete_row():
    global active_editor, main_tree

    if len(active_editor) != 0:
        active_editor[0].event_generate("<FocusOut>")

    selected_items = main_tree.selection()
    if selected_items:
        for item in selected_items:
            main_tree.delete(item)


# Function for adding a single row
def add_row():
    global main_tree, columns, active_editor

    if len(active_editor) != 0:
        active_editor[0].event_generate("<FocusOut>")
    
    main_tree.focus_set()

    i = len(main_tree.get_children())

    main_tree.insert("", "end", values=['' for col in columns], tags=("oddrow" if i % 2 == 0 else "evenrow"))


# Function for calling action command
def action_command(command):
    global ui_config

    if ui_config['actions'] and ui_config['actions']['enable'] and ui_config['actions'][command]:
        subprocess.call(ui_config['actions'][command], shell=True)


# Function for decorate or styling the treeview
def decorate_styling():
    # Styling
    style = ttk.Style()
    style.theme_use("clam")
    style.configure("Treeview", rowheight=40) 
    style.configure("Treeview.Heading", font=("Helvetica", 11, "bold"))



# Function for building the main_tree
def build_tree():
    global main_tree, root, columns

    root = tk.Tk()
    root.title("Manage crypto checker jobs")
    columns = ('email', 'source_coin', 'target_coin', 'source_value', 'target_value', 'comparison', 'email_sent_count')

    main_tree = ttk.Treeview(root, columns=columns, show="headings")

    main_tree.tag_configure("oddrow", background="#FAFAFA")
    main_tree.tag_configure("evenrow", background="#F7F7F7")

    main_tree.pack(fill="both", expand=True, padx=5, pady=5)
    main_tree.bind("<Double-1>", lambda event: edit_cell(event))


# Function for building rows
def build_rows():
    global rows, main_tree, columns

    with open('config.json', 'r') as f:
        worker_config = json.load(f)
        rows = worker_config['jobs']

    for row in main_tree.get_children():
        main_tree.delete(row)

    # Build Rows
    for i, item in enumerate(rows):
        main_tree.insert("", "end", values=[item[col] for col in columns], tags=("oddrow" if i % 2 == 0 else "evenrow"))


# Function for building headings
def build_headings():
    global main_tree

    main_tree.heading("email", text="Email")
    main_tree.heading("source_coin", text="Source Coin")
    main_tree.heading("target_coin", text="Target Coin")
    main_tree.heading("source_value", text="Source Value")
    main_tree.heading("target_value", text="Target Value")
    main_tree.heading("comparison", text="Comparison")
    main_tree.heading("email_sent_count", text="Email Sent Count")


# function for building Buttons
def build_buttons():
    global root

    button_frame = tk.Frame(root, borderwidth=2)
    button_frame.pack()

    add_button = tk.Button(button_frame, text="Add", command=lambda: add_row())
    add_button.pack(side=tk.LEFT, padx=5, pady=5)

    delete_button = tk.Button(button_frame, text="Delete", command=lambda: delete_row())
    delete_button.pack(side=tk.LEFT, padx=5, pady=5)

    save_button = tk.Button(button_frame, text="Save", command=lambda: save_rows())
    save_button.pack(side=tk.LEFT, padx=5, pady=5)

    reload_button = tk.Button(button_frame, text="Reload", command=lambda: build_rows())
    reload_button.pack(side=tk.LEFT, padx=5, pady=5)

    if ui_config['actions'] and ui_config['actions']['enable'] and ui_config['actions']['push']:
        push_button = tk.Button(button_frame, text="Upload to server", command=lambda: action_command('push'))
        push_button.pack(side=tk.LEFT, padx=5, pady=5)

    if ui_config['actions'] and ui_config['actions']['enable'] and ui_config['actions']['pull']:
        pull_button = tk.Button(button_frame, text="Retrieve from server", command=lambda: action_command('pull'))
        pull_button.pack(side=tk.LEFT, padx=5, pady=5)



# Load the config
def load_config():
    global ui_config

    with open('configui.json', 'r') as f:
    ui_config = json.load(f)



# Main function
def main():

    global root
    
    # Loading the config
    load_config()

    # Style the table
    decorate_styling()

    # Build the main treeview
    build_tree()

    # Build headings
    build_headings()

    # Build Rows
    build_rows()

    # Build Buttons
    build_buttons()

    root.mainloop()


# Boot the UI
if __name__ == '__main__':
    main()
