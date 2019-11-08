import React from 'react'

export default function NavBar() {
    return (
        <nav id="my-navbar" className="navbar navbar-expand-sm sticky-top">
        <ul className="navbar-nav">
            <li className="nav-item">
                <a className="nav-link" href="#">Link 1</a>
            </li>
            <li className="nav-item">
                <a className="nav-link" href="#">Link 2</a>
            </li>
            <li className="nav-item">
                <a className="nav-link" href="#">Link 3</a>
            </li>
        </ul>
        </nav>
    )
}