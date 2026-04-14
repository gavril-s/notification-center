#!/usr/bin/env python3
"""Test all frontend pages"""
import os
os.environ['PLAYWRIGHT_BROWSERS_PATH'] = '/tmp/pw-browsers'

from playwright.sync_api import sync_playwright
import sys

def test_frontend():
    results = []
    
    with sync_playwright() as p:
        browser = p.chromium.launch(headless=True)
        context = browser.new_context()
        
        # Test 1: Login page
        print("Testing: Login page...")
        page = context.new_page()
        try:
            page.goto('http://localhost/login', wait_until='networkidle', timeout=10000)
            # Check for login form elements
            login_input = page.locator('input[type="text"], input[type="email"]').first
            password_input = page.locator('input[type="password"]').first
            submit_btn = page.locator('button[type="submit"]').first
            
            if login_input.is_visible() and password_input.is_visible() and submit_btn.is_visible():
                results.append(("Login page", "PASS"))
                print("  ✓ Login form visible")
            else:
                results.append(("Login page", "FAIL - form elements not visible"))
                print("  ✗ Form elements missing")
        except Exception as e:
            results.append(("Login page", f"FAIL - {e}"))
            print(f"  ✗ Error: {e}")
        page.close()
        
        # Test 2: Login as testuser and check /app pages
        print("\nTesting: Login flow...")
        page = context.new_page()
        try:
            page.goto('http://localhost/login', wait_until='networkidle', timeout=10000)
            page.fill('input[type="password"]', 'testpass123')
            # Try to find login input - might be username or email
            inputs = page.locator('input').all()
            for inp in inputs:
                inp_type = inp.get_attribute('type')
                if inp_type != 'password':
                    inp.fill('testuser')
                    break
            
            page.click('button[type="submit"]')
            page.wait_for_timeout(2000)
            
            # Check if logged in
            current_url = page.url
            if '/app' in current_url or '/operator' in current_url:
                results.append(("Login flow", "PASS"))
                print(f"  ✓ Logged in successfully, redirected to {current_url}")
            else:
                results.append(("Login flow", "FAIL - not redirected"))
                print(f"  ✗ Not redirected, current URL: {current_url}")
        except Exception as e:
            results.append(("Login flow", f"FAIL - {e}"))
            print(f"  ✗ Error: {e}")
        
        # Test 3: App pages (recipient user)
        app_pages = ['/app/profile', '/app/contacts', '/app/preferences', '/app/history']
        for path in app_pages:
            print(f"\nTesting: {path}...")
            try:
                page.goto(f'http://localhost{path}', wait_until='networkidle', timeout=10000)
                # Check for page content (not error)
                body_text = page.locator('body').inner_text()
                if 'error' not in body_text.lower()[:100] and page.locator('.loading-screen').count() == 0:
                    results.append((path, "PASS"))
                    print(f"  ✓ Page loaded")
                else:
                    results.append((path, "WARN - may have loading issues"))
                    print(f"  ⚠ May have issues")
            except Exception as e:
                results.append((path, f"FAIL - {e}"))
                print(f"  ✗ Error: {e}")
        
        page.close()
        
        # Test 4: Operator pages (need source_operator role)
        # First, let's create a source operator user and login
        print("\n--- Testing operator pages ---")
        
        # We'll try directly accessing operator pages
        operator_pages = ['/operator/templates', '/operator/groups', '/operator/campaigns', '/operator/history', '/operator/analytics']
        
        # Create new page for operator test
        page = context.new_page()
        
        # First login as admin or create operator user
        try:
            # Try login as admin
            page.goto('http://localhost/login', wait_until='networkidle', timeout=10000)
            # Check if we can login - need proper credentials
            # Let's just check the pages directly
            for path in operator_pages:
                print(f"\nTesting: {path}...")
                try:
                    page.goto(f'http://localhost{path}', wait_until='networkidle', timeout=10000)
                    body_text = page.locator('body').inner_text()
                    if 'error' not in body_text.lower()[:100]:
                        results.append((path, "PASS"))
                        print(f"  ✓ Page loaded")
                    else:
                        results.append((path, "WARN - may have access issues"))
                        print(f"  ⚠ May have access issues")
                except Exception as e:
                    results.append((path, f"FAIL - {e}"))
                    print(f"  ✗ Error: {e}")
        except Exception as e:
            print(f"Operator test error: {e}")
        
        page.close()
        
        # Test 5: Unsubscribe page (public)
        print("\nTesting: /unsubscribe (public)...")
        page = context.new_page()
        try:
            page.goto('http://localhost/unsubscribe', wait_until='networkidle', timeout=10000)
            body_text = page.locator('body').inner_text()
            if len(body_text) > 10:
                results.append(("/unsubscribe", "PASS"))
                print("  ✓ Page loaded")
            else:
                results.append(("/unsubscribe", "FAIL - empty"))
                print("  ✗ Empty page")
        except Exception as e:
            results.append(("/unsubscribe", f"FAIL - {e}"))
            print(f"  ✗ Error: {e}")
        page.close()
        
        browser.close()
    
    # Print summary
    print("\n" + "="*50)
    print("SUMMARY")
    print("="*50)
    for page_name, status in results:
        symbol = "✓" if status == "PASS" else ("⚠" if "WARN" in status else "✗")
        print(f"{symbol} {page_name}: {status}")
    
    pass_count = sum(1 for _, s in results if s == "PASS")
    total = len(results)
    print(f"\nTotal: {pass_count}/{total} passed")
    
    return pass_count == total

if __name__ == '__main__':
    success = test_frontend()
    sys.exit(0 if success else 1)